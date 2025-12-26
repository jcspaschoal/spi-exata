package tenantdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/domain/tenantbus"
	"github.com/jcpaschoal/spi-exata/business/sdk/sqldb"
	"github.com/jcpaschoal/spi-exata/foundatiton/logger"
	"github.com/jmoiron/sqlx"
)

// Store manages the set of APIs for tenant database access.
type Store struct {
	log *logger.Logger
	db  *sqlx.DB
}

// NewStore constructs the api for data access.
func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{
		log: log,
		db:  db,
	}
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
		public.tenant
	WHERE
		slug = :slug`

	// Struct interna para mapeamento do resultado
	var result struct {
		ID uuid.UUID `db:"tenant_id"`
	}

	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &result); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return uuid.Nil, tenantbus.ErrNotFound
		}
		return uuid.Nil, fmt.Errorf("querying tenant id by slug: %w", err)
	}

	return result.ID, nil
}
