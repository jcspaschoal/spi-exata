package tenantdb

import (
	"time"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/domain/tenantbus"
)

// tenantDB represents the structure of the tenant table in the database.
type tenantDB struct {
	ID        uuid.UUID `db:"tenant_id"`
	Name      string    `db:"name"`
	Slug      string    `db:"slug"`
	Enabled   bool      `db:"enabled"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func toDBTenant(bus tenantbus.Tenant) tenantDB {
	return tenantDB{
		ID:        bus.ID,
		Name:      bus.Name,
		Slug:      bus.Slug,
		Enabled:   bus.Enabled,
		CreatedAt: bus.CreatedAt,
		UpdatedAt: bus.UpdatedAt,
	}
}

func toBusTenant(db tenantDB) tenantbus.Tenant {
	return tenantbus.Tenant{
		ID:        db.ID,
		Name:      db.Name,
		Slug:      db.Slug,
		Enabled:   db.Enabled,
		CreatedAt: db.CreatedAt,
		UpdatedAt: db.UpdatedAt,
	}
}
