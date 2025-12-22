package acldb

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/domain/aclbus"
	"github.com/jcpaschoal/spi-exata/business/sdk/sqldb"
	"github.com/jcpaschoal/spi-exata/business/sdk/sqldb/dbarray"
	"github.com/jcpaschoal/spi-exata/business/types/role"
	"github.com/jcpaschoal/spi-exata/foundatiton/logger"
	"github.com/jmoiron/sqlx"
)

type Store struct {
	log *logger.Logger
	db  sqlx.ExtContext
}

// NewStore constructs the api for data access.
func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{
		log: log,
		db:  db,
	}
}

func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (aclbus.Store, error) {
	ec, err := sqldb.GetExtContext(tx)
	if err != nil {
		return nil, err
	}

	store := Store{
		log: s.log,
		db:  ec,
	}

	return &store, nil
}

func (s *Store) Create(ctx context.Context, userID uuid.UUID, nw aclbus.NewACL) error {
	const q = `
	INSERT INTO "public"."acl"
		(resource_id, user_id, resource_type_id, actions)
	VALUES
		(
			:resource_id, 
			:user_id, 
			(SELECT resource_type_id FROM "public"."resource_type" WHERE name = :resource_type), 
			:actions::actions_enum[]
		)`

	dbACL := toDBACL(userID, nw.ResourceID, nw.Resource, nw.Actions)

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, dbACL); err != nil {
		var dupErr sqldb.ErrDBDuplicatedEntry
		if errors.As(err, &dupErr) {
			return fmt.Errorf("namedexeccontext: %w", aclbus.ErrUnique)
		}
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID) error {
	const q = `
	DELETE FROM
		"public"."acl"
	WHERE
		user_id = :user_id AND resource_id = :resource_id`

	data := struct {
		UserID     uuid.UUID `db:"user_id"`
		ResourceID uuid.UUID `db:"resource_id"`
	}{
		UserID:     userID,
		ResourceID: resourceID,
	}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Update(ctx context.Context, userID uuid.UUID, p aclbus.ACLUpdate) error {
	const q = `
	UPDATE
		"public"."acl"
	SET
		actions = :actions::actions_enum[]
	WHERE
		user_id = :user_id 
		AND resource_id = :resource_id
		AND resource_type_id = (SELECT resource_type_id FROM "public"."resource_type" WHERE name = :resource_type)`

	actsStr := make([]string, len(p.Actions))
	for i, act := range p.Actions {
		actsStr[i] = act.String()
	}

	data := struct {
		UserID       uuid.UUID      `db:"user_id"`
		ResourceID   uuid.UUID      `db:"resource_id"`
		ResourceType string         `db:"resource_type"`
		Actions      dbarray.String `db:"actions"`
	}{
		UserID:       userID,
		ResourceID:   p.ResourceID,
		ResourceType: p.Resource.String(),
		Actions:      actsStr,
	}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// GetAllPermissions retrieves all ACL rules defined in the system.
func (s *Store) GetAllPermissions(ctx context.Context) ([]aclbus.ACL, error) {
	const q = `
	SELECT
		a.resource_id, a.user_id, a.actions,
		rt.name AS resource_type
	FROM
		"public"."acl" AS a
	JOIN
		"public"."resource_type" AS rt ON rt.resource_type_id = a.resource_type_id`

	var dbACLs []aclDB
	if err := sqldb.NamedQuerySlice(ctx, s.log, s.db, q, struct{}{}, &dbACLs); err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusACLs(dbACLs)
}

// ValidateAccess checks if the user has specific permissions on a concrete resource instance.
// It queries the 'acl' table directly.
// Returns nil if authorized, ErrAccessDenied if not found, or other error if DB fails.
func (s *Store) ValidateAccess(ctx context.Context, check aclbus.AccessCheck) error {
	const q = `
	SELECT
		count(1)
	FROM
		"public"."acl" AS a
	WHERE
		a.user_id = :user_id 
		AND a.resource_id = :resource_id
		AND a.resource_type_id = (SELECT resource_type_id FROM "public"."resource_type" WHERE name = :resource_type)
		AND a.actions @> :actions::text[]::actions_enum[]`

	data := struct {
		UserID       uuid.UUID `db:"user_id"`
		ResourceID   uuid.UUID `db:"resource_id"`
		ResourceType string    `db:"resource_type"`
		Actions      []string  `db:"actions"`
	}{
		UserID:       check.UserID,
		ResourceID:   check.ResourceID,
		ResourceType: check.Resource.String(),
		Actions:      []string{check.Action.String()},
	}

	var count struct {
		Count int `db:"count"`
	}

	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &count); err != nil {
		return fmt.Errorf("namedquerystruct: %w", err)
	}

	if count.Count > 0 {
		return nil
	}

	return aclbus.ErrAccessDenied
}

// ValidateAccessToResource checks if the user's role permits the requested action on the resource type.
// It queries 'users', 'role', and 'role_policy' tables to validate RBAC rules.
// It automatically grants access if the user has the 'ADMIN' role.
func (s *Store) ValidateAccessToResource(ctx context.Context, check aclbus.ResourceCheck) error {
	const q = `
	SELECT count(1)
	FROM "public"."users" u
	JOIN "public"."role" r ON r.role_id = u.role_id
	LEFT JOIN "public"."role_policy" rp ON rp.role_id = u.role_id 
		AND rp.resource_type_id = (SELECT resource_type_id FROM "public"."resource_type" WHERE name = :resource_type)
	WHERE 
		u.user_id = :user_id
		AND (
			r.name = 'ADMIN' -- Bypass: Admin role has full access
			OR
			rp.actions @> :actions::text[]::actions_enum[] -- RBAC Check
		)`

	data := struct {
		UserID       uuid.UUID `db:"user_id"`
		ResourceType string    `db:"resource_type"`
		Actions      []string  `db:"actions"`
	}{
		UserID:       check.UserID,
		ResourceType: check.Resource.String(),
		Actions:      []string{check.Action.String()},
	}

	var count struct {
		Count int `db:"count"`
	}

	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &count); err != nil {
		return fmt.Errorf("namedquerystruct: %w", err)
	}

	if count.Count > 0 {
		return nil
	}

	return aclbus.ErrAccessDenied
}

// GetAllUserRoles retrieves a map of UserID -> role.Role for all users.
func (s *Store) GetAllUserRoles(ctx context.Context) (map[uuid.UUID]role.Role, error) {
	const q = `
	SELECT 
		u.user_id, 
		r.name as role_name
	FROM 
		"public"."users" u
	JOIN 
		"public"."role" r ON r.role_id = u.role_id`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query users roles: %w", err)
	}
	defer rows.Close()

	userRoles := make(map[uuid.UUID]role.Role)

	for rows.Next() {
		var uid uuid.UUID
		var roleName string

		if err := rows.Scan(&uid, &roleName); err != nil {
			return nil, fmt.Errorf("scan user role: %w", err)
		}

		r, err := role.Parse(roleName)
		if err != nil {
			return nil, fmt.Errorf("parse role '%s': %w", roleName, err)
		}

		userRoles[uid] = r
	}

	return userRoles, nil
}

// SyncUserRole is a no-op for the database store.
// The role data is persisted by the UserBus in the 'users' table.
// Since ValidateRoleAccess queries the 'users' table directly via JOINs,
// the DB-level checks are always strongly consistent without manual sync.
func (s *Store) SyncUserRole(ctx context.Context, userID uuid.UUID, r role.Role) error {
	return nil
}
