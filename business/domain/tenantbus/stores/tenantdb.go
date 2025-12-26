package tenantdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/domain/tenantbus"
	"github.com/jcpaschoal/spi-exata/business/sdk/sqldb"
	"github.com/jcpaschoal/spi-exata/foundation/logger"
	"github.com/jmoiron/sqlx"
)

// Store manages the set of APIs for tenant database access.
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

// NewWithTx constructs a new Store value replacing the sqlx DB
// value with a sqlx DB value that is currently inside a transaction.
func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (tenantbus.Storer, error) {
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

// Create inserts a new tenant into the database.
func (s *Store) Create(ctx context.Context, t tenantbus.Tenant) error {
	const q = `
	INSERT INTO "public"."tenant"
		(tenant_id, name, slug, enabled, created_at, updated_at)
	VALUES
		(:tenant_id, :name, :slug, :enabled, :created_at, :updated_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBTenant(t)); err != nil {
		var dupErr sqldb.ErrDBDuplicatedEntry
		if errors.As(err, &dupErr) {
			if dupErr.Column == "slug" || dupErr.Column == "uq_tenant_slug" {
				return fmt.Errorf("namedexeccontext: %w", tenantbus.ErrUniqueSlug)
			}
		}
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Update replaces a tenant document in the database.
func (s *Store) Update(ctx context.Context, t tenantbus.Tenant) error {
	const q = `
	UPDATE
		"public"."tenant"
	SET 
		name = :name,
		enabled = :enabled,
		updated_at = :updated_at
	WHERE
		tenant_id = :tenant_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBTenant(t)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Delete removes a tenant from the database.
func (s *Store) Delete(ctx context.Context, t tenantbus.Tenant) error {
	const q = `
	DELETE FROM
		"public"."tenant"
	WHERE
		tenant_id = :tenant_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBTenant(t)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// QueryByID gets the specified tenant from the database.
func (s *Store) QueryByID(ctx context.Context, tenantID uuid.UUID) (tenantbus.Tenant, error) {
	data := struct {
		ID string `db:"tenant_id"`
	}{
		ID: tenantID.String(),
	}

	const q = `
	SELECT
		tenant_id, name, slug, enabled, created_at, updated_at
	FROM
		"public"."tenant"
	WHERE 
		tenant_id = :tenant_id`

	var dbT tenantDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &dbT); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return tenantbus.Tenant{}, fmt.Errorf("db: %w", tenantbus.ErrNotFound)
		}
		return tenantbus.Tenant{}, fmt.Errorf("db: %w", err)
	}

	return toBusTenant(dbT), nil
}

// QueryIDBySlug retrieves the tenant ID for the specified slug.
func (s *Store) QueryIDBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	data := struct {
		Slug string `db:"slug"`
	}{
		Slug: slug,
	}

	const q = `
	SELECT
		tenant_id
	FROM
		"public"."tenant"
	WHERE
		slug = :slug`

	var result struct {
		ID uuid.UUID `db:"tenant_id"`
	}

	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &result); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return uuid.Nil, tenantbus.ErrNotFound
		}
		return uuid.Nil, fmt.Errorf("db: %w", err)
	}

	return result.ID, nil
}

// QueryByDomain retrieves the TenantID and DashboardID associated with a specific domain.
func (s *Store) QueryByDomain(ctx context.Context, domain string) (tenantbus.TenantDashboard, error) {
	data := struct {
		Domain string `db:"domain"`
	}{
		Domain: domain,
	}

	const q = `
	SELECT
		tenant_id, dashboard_id
	FROM
		"public"."dashboard"
	WHERE
		domain = :domain`

	var result struct {
		TenantID    uuid.UUID `db:"tenant_id"`
		DashboardID uuid.UUID `db:"dashboard_id"`
	}

	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &result); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return tenantbus.TenantDashboard{}, tenantbus.ErrDomainNotFound
		}
		return tenantbus.TenantDashboard{}, fmt.Errorf("db: %w", err)
	}

	return tenantbus.TenantDashboard{
		TenantID:    result.TenantID,
		DashboardID: result.DashboardID,
	}, nil
}

// CheckTenantAccess checks if a user is a member of a tenant.
// Uses the 'tenant_membership' table where user_id is the PK.
func (s *Store) CheckTenantAccess(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) error {
	data := struct {
		UserID   string `db:"user_id"`
		TenantID string `db:"tenant_id"`
	}{
		UserID:   userID.String(),
		TenantID: tenantID.String(),
	}

	const q = `
	SELECT
		1
	FROM
		"public"."tenant_membership"
	WHERE
		user_id = :user_id AND tenant_id = :tenant_id`

	var result struct {
		Exists int `db:"?column?"`
	}

	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &result); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return tenantbus.ErrAccessDenied
		}
		return fmt.Errorf("db: %w", err)
	}

	return nil
}

// CheckUserDashboardAccess checks granular permissions for a user on a dashboard.
// Uses the composite PK (user_id, dashboard_id).
// CheckUserDashboardAccess checks granular permissions for a user on a dashboard.
// Uses the composite PK (user_id, dashboard_id) and validates the tenant context.
func (s *Store) CheckUserDashboardAccess(ctx context.Context, userID uuid.UUID, dashboardID uuid.UUID, tenantID uuid.UUID) error {
	data := struct {
		UserID      string `db:"user_id"`
		DashboardID string `db:"dashboard_id"`
		TenantID    string `db:"tenant_id"`
	}{
		UserID:      userID.String(),
		DashboardID: dashboardID.String(),
		TenantID:    tenantID.String(),
	}

	const q = `
	SELECT
		1
	FROM
		"public"."user_dashboard_access"
	WHERE
		user_id = :user_id 
		AND dashboard_id = :dashboard_id 
		AND tenant_id = :tenant_id`

	var result struct {
		Exists int `db:"?column?"`
	}

	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &result); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return tenantbus.ErrAccessDenied
		}
		return fmt.Errorf("db: %w", err)
	}

	return nil
}

// QueryTenantIDByUserID retrieves the TenantID for a user from the membership table.
func (s *Store) QueryTenantIDByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	data := struct {
		UserID string `db:"user_id"`
	}{
		UserID: userID.String(),
	}

	const q = `
	SELECT
		tenant_id
	FROM
		"public"."tenant_membership"
	WHERE
		user_id = :user_id`

	var result struct {
		TenantID uuid.UUID `db:"tenant_id"`
	}

	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &result); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			// Se não tem membership, tecnicamente não tem tenant, ou seja, user inválido neste contexto
			return uuid.Nil, tenantbus.ErrNotFound
		}
		return uuid.Nil, fmt.Errorf("db: %w", err)
	}

	return result.TenantID, nil
}

func (s *Store) QueryTenantIDByDashboardID(ctx context.Context, dashboardID uuid.UUID) (uuid.UUID, error) {
	data := struct {
		ID string `db:"dashboard_id"`
	}{
		ID: dashboardID.String(),
	}

	const q = `
	SELECT tenant_id
	FROM "public"."dashboard"
	WHERE dashboard_id = :dashboard_id`

	var result struct {
		TenantID uuid.UUID `db:"tenant_id"`
	}

	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &result); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return uuid.Nil, tenantbus.ErrNotFound
		}
		return uuid.Nil, fmt.Errorf("db: %w", err)
	}

	return result.TenantID, nil
}

// AddUserToTenant inserts a record into tenant_membership.
func (s *Store) AddUserToTenant(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) error {
	data := struct {
		UserID   string `db:"user_id"`
		TenantID string `db:"tenant_id"`
	}{
		UserID:   userID.String(),
		TenantID: tenantID.String(),
	}

	// ON CONFLICT DO NOTHING garante idempotência se rodar o utilitário 2 vezes
	const q = `
	INSERT INTO "public"."tenant_membership" (user_id, tenant_id, created_at)
	VALUES (:user_id, :tenant_id, NOW())
	ON CONFLICT (user_id) DO UPDATE SET tenant_id = EXCLUDED.tenant_id`
	// Nota: Como a PK é user_id (1:1 strict), atualizamos o tenant se o usuário já tiver um.

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// AddUserToDashboard inserts a record into user_dashboard_access.
func (s *Store) AddUserToDashboard(ctx context.Context, userID uuid.UUID, dashboardID uuid.UUID, tenantID uuid.UUID) error {
	data := struct {
		UserID      string `db:"user_id"`
		DashboardID string `db:"dashboard_id"`
		TenantID    string `db:"tenant_id"`
	}{
		UserID:      userID.String(),
		DashboardID: dashboardID.String(),
		TenantID:    tenantID.String(),
	}

	const q = `
	INSERT INTO "public"."user_dashboard_access" (user_id, dashboard_id, tenant_id, created_at)
	VALUES (:user_id, :dashboard_id, :tenant_id, NOW())
	ON CONFLICT (user_id, dashboard_id) DO NOTHING`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}
