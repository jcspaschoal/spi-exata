package tenantbus

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/foundatiton/logger"
)

// ErrNotFound is returned when a tenant is not found.
var ErrNotFound = errors.New("tenant not found")

// Storer defines the behavior required by the tenantbus to interact with the database.
type Storer interface {
	QueryIDBySlug(ctx context.Context, slug string) (uuid.UUID, error)
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

// QueryIDBySlug returns the tenant ID for the specified slug string.
func (c *Core) QueryIDBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	id, err := c.storer.QueryIDBySlug(ctx, slug)
	if err != nil {
		return uuid.Nil, fmt.Errorf("query by slug: %w", err)
	}

	return id, nil
}
