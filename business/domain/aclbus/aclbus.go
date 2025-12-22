package aclbus

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/sdk/sqldb"
	"github.com/jcpaschoal/spi-exata/business/types/role"
	"github.com/jcpaschoal/spi-exata/foundatiton/otel"
)

var (
	ErrUnique       = errors.New("acl entry already exists")
	ErrAccessDenied = errors.New("access denied")
)

type Store interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Store, error)
	Create(ctx context.Context, userID uuid.UUID, nw NewACL) error
	Delete(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID) error
	Update(ctx context.Context, userID uuid.UUID, p ACLUpdate) error
	GetAllPermissions(ctx context.Context) ([]ACL, error)
	ValidateAccess(ctx context.Context, check AccessCheck) error
	ValidateAccessToResource(ctx context.Context, check ResourceCheck) error
	GetAllUserRoles(ctx context.Context) (map[uuid.UUID]role.Role, error)

	// SyncUserRole updates the user's role in the cache to match the new state in UserBus.
	// This ensures the in-memory ACL decision engine reflects role changes immediately.
	SyncUserRole(ctx context.Context, userID uuid.UUID, r role.Role) error
}

type Core struct {
	store Store
}

func NewCore(store Store) *Core {
	return &Core{
		store: store,
	}
}

func (c *Core) NewWithTx(tx sqldb.CommitRollbacker) (*Core, error) {
	storer, err := c.store.NewWithTx(tx)
	if err != nil {
		return nil, err
	}

	nc := NewCore(storer)
	return nc, nil
}

func (c *Core) Create(ctx context.Context, userID uuid.UUID, nw NewACL) error {
	ctx, span := otel.AddSpan(ctx, "business.aclbus.create")
	defer span.End()

	if err := c.store.Create(ctx, userID, nw); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	return nil
}

func (c *Core) Delete(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID) error {
	ctx, span := otel.AddSpan(ctx, "business.aclbus.delete")
	defer span.End()

	if err := c.store.Delete(ctx, userID, resourceID); err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	return nil
}

func (c *Core) Update(ctx context.Context, userID uuid.UUID, newRule ACLUpdate) error {
	ctx, span := otel.AddSpan(ctx, "business.aclbus.update")
	defer span.End()

	if err := c.store.Update(ctx, userID, newRule); err != nil {
		return fmt.Errorf("update: %w", err)
	}

	return nil
}

func (c *Core) GetAllPermissions(ctx context.Context) ([]ACL, error) {
	ctx, span := otel.AddSpan(ctx, "business.aclbus.getAllPermissions")
	defer span.End()

	acls, err := c.store.GetAllPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("getAllPermissions: %w", err)
	}

	return acls, nil
}

// ValidateAccess verifies permission for a specific Instance (e.g. ID 123).
func (c *Core) ValidateAccess(ctx context.Context, check AccessCheck) error {
	ctx, span := otel.AddSpan(ctx, "business.aclbus.validateAccess")
	defer span.End()

	if err := c.store.ValidateAccess(ctx, check); err != nil {
		return fmt.Errorf("validateAccess: %w", err)
	}
	return nil
}

// ValidateAccessToResource verifies permission for a Resource Type (e.g. Can Create Dashboard?).
func (c *Core) ValidateAccessToResource(ctx context.Context, check ResourceCheck) error {
	ctx, span := otel.AddSpan(ctx, "business.aclbus.validateAccessToResource")
	defer span.End()

	if err := c.store.ValidateAccessToResource(ctx, check); err != nil {
		return fmt.Errorf("validateAccessToResource: %w", err)
	}
	return nil
}

// SyncUserRole updates the cache when a user's role changes in the system.
func (c *Core) SyncUserRole(ctx context.Context, userID uuid.UUID, r role.Role) error {
	ctx, span := otel.AddSpan(ctx, "business.aclbus.syncUserRole")
	defer span.End()

	if err := c.store.SyncUserRole(ctx, userID, r); err != nil {
		return fmt.Errorf("syncUserRole: %w", err)
	}

	return nil
}
