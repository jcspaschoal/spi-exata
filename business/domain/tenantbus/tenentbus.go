package tenantbus

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/sdk/sqldb"
	"github.com/jcpaschoal/spi-exata/foundatiton/logger"
	"github.com/jcpaschoal/spi-exata/foundatiton/otel"
)

var (
	ErrNotFound       = errors.New("tenant not found")
	ErrDomainNotFound = errors.New("domain not found")
	ErrAccessDenied   = errors.New("access denied")
	ErrUniqueSlug     = errors.New("slug is not unique")
)

// Storer defines the behavior required by the tenantbus to interact with the database.
type Storer interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Storer, error)

	// Administrative
	Create(ctx context.Context, t Tenant) error
	Update(ctx context.Context, t Tenant) error
	Delete(ctx context.Context, t Tenant) error
	QueryByID(ctx context.Context, tenantID uuid.UUID) (Tenant, error)

	QueryIDBySlug(ctx context.Context, slug string) (uuid.UUID, error)
	QueryByDomain(ctx context.Context, domain string) (TenantDashboard, error)

	CheckTenantAccess(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) error
	CheckUserDashboardAccess(ctx context.Context, userID uuid.UUID, dashboardID uuid.UUID, tenantID uuid.UUID) error
	QueryTenantIDByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

// Core manages the set of APIs for tenant access.
type Core struct {
	storer Storer
	log    *logger.Logger
}

// NewCore constructs a core for tenant api access.
func NewCore(log *logger.Logger, storer Storer) *Core {
	return &Core{
		storer: storer,
		log:    log,
	}
}

// NewWithTx constructs a new Core value replacing the Storer
// value with a Storer value that is currently inside a transaction.
func (c *Core) NewWithTx(tx sqldb.CommitRollbacker) (*Core, error) {
	storer, err := c.storer.NewWithTx(tx)
	if err != nil {
		return nil, fmt.Errorf("newWithTx: %w", err)
	}

	return NewCore(c.log, storer), nil
}

// Create adds a new tenant to the system.
func (c *Core) Create(ctx context.Context, nt NewTenant) (Tenant, error) {
	ctx, span := otel.AddSpan(ctx, "business.tenantbus.create")
	defer span.End()

	now := time.Now()

	t := Tenant{
		ID:        uuid.New(),
		Name:      nt.Name,
		Slug:      nt.Slug,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := c.storer.Create(ctx, t); err != nil {
		return Tenant{}, fmt.Errorf("create: %w", err)
	}

	return t, nil
}

// Update modifies data about a tenant.
func (c *Core) Update(ctx context.Context, t Tenant, ut UpdateTenant) (Tenant, error) {
	ctx, span := otel.AddSpan(ctx, "business.tenantbus.update")
	defer span.End()

	if ut.Name != nil {
		t.Name = *ut.Name
	}

	if ut.Enabled != nil {
		t.Enabled = *ut.Enabled
	}

	t.UpdatedAt = time.Now()

	if err := c.storer.Update(ctx, t); err != nil {
		return Tenant{}, fmt.Errorf("update: %w", err)
	}

	return t, nil
}

// Delete removes the specified tenant from the system.
func (c *Core) Delete(ctx context.Context, t Tenant) error {
	ctx, span := otel.AddSpan(ctx, "business.tenantbus.delete")
	defer span.End()

	if err := c.storer.Delete(ctx, t); err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	return nil
}

// QueryByID finds the tenant by the specified ID.
func (c *Core) QueryByID(ctx context.Context, tenantID uuid.UUID) (Tenant, error) {
	ctx, span := otel.AddSpan(ctx, "business.tenantbus.queryByID")
	defer span.End()

	tenant, err := c.storer.QueryByID(ctx, tenantID)
	if err != nil {
		return Tenant{}, fmt.Errorf("query: tenantID[%s]: %w", tenantID, err)
	}

	return tenant, nil
}

// QueryIDBySlug returns the tenant ID for the specified slug string.
func (c *Core) QueryIDBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	ctx, span := otel.AddSpan(ctx, "business.tenantbus.queryIDBySlug")
	defer span.End()

	id, err := c.storer.QueryIDBySlug(ctx, slug)
	if err != nil {
		return uuid.Nil, fmt.Errorf("query by slug[%s]: %w", slug, err)
	}

	return id, nil
}

// ResolveDomain translates a domain string (e.g. "sales.corp.com") into the corresponding
// TenantDashboard context (TenantID and DashboardID).
func (c *Core) ResolveDomain(ctx context.Context, domain string) (TenantDashboard, error) {
	ctx, span := otel.AddSpan(ctx, "business.tenantbus.resolveDomain")
	defer span.End()

	td, err := c.storer.QueryByDomain(ctx, domain)

	if err != nil {
		return TenantDashboard{}, fmt.Errorf("queryByDomain[%s]: %w", domain, err)
	}
	return td, nil
}

// CheckAccess checks if the user is a member of the tenant.
// Returns nil if allowed, error if denied.
func (c *Core) CheckAccess(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) error {
	ctx, span := otel.AddSpan(ctx, "business.tenantbus.checkAccess")
	defer span.End()

	if err := c.storer.CheckTenantAccess(ctx, userID, tenantID); err != nil {
		return fmt.Errorf("checkTenantAccess: %w", err)
	}

	return nil
}

// QueryTenantIDByUserID retrieves the TenantID associated with a specific UserID.
// This is critical for authentication to ensure 1 User = 1 Tenant strict compliance.
func (c *Core) QueryTenantIDByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	ctx, span := otel.AddSpan(ctx, "business.tenantbus.queryTenantIDByUserID")
	defer span.End()

	tenantID, err := c.storer.QueryTenantIDByUserID(ctx, userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("queryTenantIDByUserID[%s]: %w", userID, err)
	}

	return tenantID, nil
}

// AuthorizeUserAccessToDashboard checks if a specific user has granular permission to view a dashboard.
// It orchestrates the flow: Resolve Domain -> Check Access.
// Return TenantDashboard on success (useful for JWT generation).
func (c *Core) AuthorizeUserAccessToDashboard(ctx context.Context, userID uuid.UUID, domain string) (TenantDashboard, error) {
	ctx, span := otel.AddSpan(ctx, "business.tenantbus.canUserAccessDashboard")
	defer span.End()

	// 1. Resolve Domain to get internal IDs (TenantID, DashboardID)
	td, err := c.ResolveDomain(ctx, domain)
	if err != nil {
		return TenantDashboard{}, fmt.Errorf("resolveDomain: %w", err)
	}

	// 2. Perform the granular check using the IDs
	if err := c.storer.CheckUserDashboardAccess(ctx, userID, td.DashboardID, td.TenantID); err != nil {
		return TenantDashboard{}, fmt.Errorf("checkUserDashboardAccess[%s]: %w", userID, err)
	}
	return td, nil
}
