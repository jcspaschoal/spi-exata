package dashboarddb

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/domain/dashboardbus"
	"github.com/jcpaschoal/spi-exata/business/types/name"
)

type dashboardDB struct {
	ID        uuid.UUID      `db:"dashboard_id"`
	TenantID  uuid.UUID      `db:"tenant_id"`
	Name      string         `db:"name"`
	Domain    sql.NullString `db:"domain"`
	Logo      []byte         `db:"logo"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

func toDBDashboard(bus dashboardbus.Dashboard) dashboardDB {
	var domain sql.NullString
	if bus.Domain != nil {
		domain = sql.NullString{String: *bus.Domain, Valid: true}
	}

	return dashboardDB{
		ID:        bus.ID,
		TenantID:  bus.TenantID,
		Name:      bus.Name.String(),
		Domain:    domain,
		Logo:      bus.Logo,
		CreatedAt: bus.CreatedAt.UTC(),
		UpdatedAt: bus.UpdatedAt.UTC(),
	}
}

func toBusDashboard(db dashboardDB) (dashboardbus.Dashboard, error) {
	var domain *string
	if db.Domain.Valid {
		domain = &db.Domain.String
	}

	n, err := name.Parse(db.Name)
	if err != nil {
		return dashboardbus.Dashboard{}, fmt.Errorf("parse name: %w", err)
	}

	return dashboardbus.Dashboard{
		ID:        db.ID,
		TenantID:  db.TenantID,
		Name:      n,
		Domain:    domain,
		Logo:      db.Logo,
		CreatedAt: db.CreatedAt.In(time.Local),
		UpdatedAt: db.UpdatedAt.In(time.Local),
	}, nil
}
