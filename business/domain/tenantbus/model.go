package tenantbus

import (
	"time"

	"github.com/google/uuid"
)

// Tenant represents a client organization or workspace in the system.
type Tenant struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserDashboardAccess represents the granular permission link between a user and a dashboard.
type TenantDashboard struct {
	TenantID    uuid.UUID
	DashboardID uuid.UUID
}

// NewTenant contains information needed to create a new tenant.
type NewTenant struct {
	Name string
	Slug string
}

// UpdateTenant contains information needed to update a tenant.
type UpdateTenant struct {
	Name    *string
	Enabled *bool
}
