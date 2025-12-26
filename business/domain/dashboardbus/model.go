package dashboardbus

import (
	"time"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/types/name"
)

// Dashboard represents the dashboard entity in the system.
type Dashboard struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Name      name.Name // Using custom Name type as requested
	Domain    *string
	Logo      []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewDashboard contains information needed to create a new dashboard.
type NewDashboard struct {
	TenantID uuid.UUID
	Name     name.Name
	Domain   *string
	Logo     []byte
}

// UpdateDashboard contains information needed to update a dashboard.
type UpdateDashboard struct {
	Name   *name.Name
	Domain *string
	Logo   []byte
}
