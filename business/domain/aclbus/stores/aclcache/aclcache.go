package aclcache

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/domain/aclbus"
	"github.com/jcpaschoal/spi-exata/business/sdk/sqldb"
	"github.com/jcpaschoal/spi-exata/business/types/role"
	"github.com/jcpaschoal/spi-exata/foundatiton/logger"
)

// Store implements the aclbus.Store interface with a Write-Through Cache strategy.
type Store struct {
	log    *logger.Logger
	storer aclbus.Store // The real Database Store (acldb)
	cache  *memoryCache // The isolated Casbin logic
}

// NewStore constructs the cached store.
func NewStore(log *logger.Logger, storer aclbus.Store) (*Store, error) {
	// 1. Initialize Memory Cache (Casbin)
	mem, err := newMemoryCache(log)
	if err != nil {
		return nil, err
	}

	s := &Store{
		log:    log,
		storer: storer,
		cache:  mem,
	}

	// 2. Initial Sync (Warm-up)
	// We use Background context here because this runs at startup, outside of any request.
	if err := s.syncCache(context.Background()); err != nil {
		return nil, fmt.Errorf("sync cache: %w", err)
	}

	return s, nil
}

func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (aclbus.Store, error) {
	newStorer, err := s.storer.NewWithTx(tx)
	if err != nil {
		return nil, err
	}

	return &Store{
		log:    s.log,
		storer: newStorer,
		cache:  s.cache,
	}, nil
}

func (s *Store) Create(ctx context.Context, userID uuid.UUID, nw aclbus.NewACL) error {
	if err := s.storer.Create(ctx, userID, nw); err != nil {
		return err
	}

	for _, act := range nw.Actions {
		s.cache.add(ctx, userID, nw.Resource, nw.ResourceID, act)
	}

	return nil
}

func (s *Store) Update(ctx context.Context, userID uuid.UUID, p aclbus.ACLUpdate) error {
	if err := s.storer.Update(ctx, userID, p); err != nil {
		return err
	}

	s.cache.clearInstanceRules(ctx, userID, p.ResourceID)

	for _, act := range p.Actions {
		s.cache.add(ctx, userID, p.Resource, p.ResourceID, act)
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID) error {
	if err := s.storer.Delete(ctx, userID, resourceID); err != nil {
		return err
	}

	s.cache.clearInstanceRules(ctx, userID, resourceID)

	return nil
}

// ValidateAccess verifies permission for a specific Instance (ACL).
func (s *Store) ValidateAccess(ctx context.Context, check aclbus.AccessCheck) error {
	// 1. Memory Check
	if err := s.cache.check(ctx, check.UserID, check.Resource, check.ResourceID, check.Action); err == nil {
		return nil
	}

	// 2. DB Check (Fallback)
	if err := s.storer.ValidateAccess(ctx, check); err != nil {
		return err
	}

	// 3. Self-Repair
	s.log.Info(ctx, "aclcache: cache miss/repair triggered", "user_id", check.UserID, "resource", check.ResourceID)
	s.cache.add(ctx, check.UserID, check.Resource, check.ResourceID, check.Action)

	return nil
}

// ValidateAccessToResource verifies permission for a Resource Type (RBAC).
func (s *Store) ValidateAccessToResource(ctx context.Context, check aclbus.ResourceCheck) error {
	// 1. Memory Check (RBAC Only)
	if err := s.cache.check(ctx, check.UserID, check.Resource, uuid.Nil, check.Action); err == nil {
		return nil
	}

	// 2. DB Check (Fallback)
	if err := s.storer.ValidateAccessToResource(ctx, check); err != nil {
		return err
	}

	s.log.Info(ctx, "aclcache: rbac cache miss/repair triggered", "user_id", check.UserID, "resource_type", check.Resource)

	s.cache.add(ctx, check.UserID, check.Resource, uuid.Nil, check.Action)

	return nil
}

// SyncUserRole updates the Casbin memory to reflect the user's new role immediately.
func (s *Store) SyncUserRole(ctx context.Context, userID uuid.UUID, r role.Role) error {
	if err := s.storer.SyncUserRole(ctx, userID, r); err != nil {
		return err
	}

	s.cache.setUserRole(ctx, userID, r)

	s.log.Info(ctx, "aclcache: user role synced in memory", "user_id", userID, "new_role", r.String())

	return nil
}

func (s *Store) GetAllPermissions(ctx context.Context) ([]aclbus.ACL, error) {
	return s.storer.GetAllPermissions(ctx)
}

func (s *Store) GetAllUserRoles(ctx context.Context) (map[uuid.UUID]role.Role, error) {
	return s.storer.GetAllUserRoles(ctx)
}

func (s *Store) syncCache(ctx context.Context) error {
	userRoles, err := s.storer.GetAllUserRoles(ctx)
	if err != nil {
		return fmt.Errorf("fetch user roles: %w", err)
	}

	s.cache.loadRoles(ctx, userRoles)

	acls, err := s.storer.GetAllPermissions(ctx)
	if err != nil {
		return fmt.Errorf("fetch permissions: %w", err)
	}

	for _, rule := range acls {
		for _, act := range rule.Actions {
			s.cache.add(ctx, rule.UserID, rule.Resource, rule.ResourceID, act)
		}
	}

	return nil
}
