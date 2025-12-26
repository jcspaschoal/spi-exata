package dashboarddb

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/domain/dashboardbus"
	"github.com/jcpaschoal/spi-exata/business/sdk/sqldb"
	"github.com/jcpaschoal/spi-exata/foundation/logger"
	"github.com/jmoiron/sqlx"
)

// Store manages the set of APIs for dashboard database access.
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
func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (dashboardbus.Storer, error) {
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

// Create inserts a new dashboard into the database.
// It uses a CTE to atomically create the Resource (parent) and the Dashboard (child).
func (s *Store) Create(ctx context.Context, d dashboardbus.Dashboard) (dashboardbus.Dashboard, error) {
	// 1 = ResourceType DASHBOARD
	const q = `
	WITH new_resource AS (
		INSERT INTO "public"."resource" (resource_type_id)
		VALUES (1)
		RETURNING resource_id
	)
	INSERT INTO "public"."dashboard"
		(dashboard_id, tenant_id, name, domain, logo, created_at, updated_at)
	SELECT
		resource_id, :tenant_id, :name, :domain, :logo, :created_at, :updated_at
	FROM
		new_resource
	RETURNING dashboard_id`

	var result struct {
		ID uuid.UUID `db:"dashboard_id"`
	}

	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, toDBDashboard(d), &result); err != nil {
		return dashboardbus.Dashboard{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	d.ID = result.ID
	return d, nil
}

// QueryByID gets the specified dashboard from the database.
func (s *Store) QueryByID(ctx context.Context, dashboardID uuid.UUID) (dashboardbus.Dashboard, error) {
	data := struct {
		ID string `db:"dashboard_id"`
	}{
		ID: dashboardID.String(),
	}

	const q = `
	SELECT
		dashboard_id, tenant_id, name, domain, logo, created_at, updated_at
	FROM
		"public"."dashboard"
	WHERE 
		dashboard_id = :dashboard_id`

	var dbDash dashboardDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &dbDash); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return dashboardbus.Dashboard{}, fmt.Errorf("db: %w", dashboardbus.ErrNotFound)
		}
		return dashboardbus.Dashboard{}, fmt.Errorf("db: %w", err)
	}

	return toBusDashboard(dbDash)
}

// Update replaces a dashboard record in the database.
func (s *Store) Update(ctx context.Context, d dashboardbus.Dashboard) error {
	const q = `
	UPDATE
		"public"."dashboard"
	SET 
		name = :name,
		domain = :domain,
		logo = :logo,
		updated_at = :updated_at
	WHERE
		dashboard_id = :dashboard_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBDashboard(d)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}
