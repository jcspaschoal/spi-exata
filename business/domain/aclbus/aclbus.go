package aclbus

import (
	"context"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/foundatiton/otel"
)

type Store interface {
	Create(ctx context.Context, userID uuid.UUID, nw NewACL) error
	Delete(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID) error
	Update(ctx context.Context, userID uuid.UUID, p ACLUpdate) error
	GetAllPermissions(ctx context.Context) ([]ACL, error)
	ValidateAccess(ctx context.Context, permission ACL) (bool, error)
}

type Core struct {
	store Store
}

func NewCore(store Store) *Core {
	return &Core{
		store: store,
	}
}

func (c *Core) Create(ctx context.Context, userID uuid.UUID, nw NewACL) error {

	ctx, span := otel.AddSpan(ctx, "business.aclbus.create")
	defer span.End()

	return nil
}

func (c *Core) Delete(ctx context.Context, userID uuid.UUID, nw NewACL) error {

	ctx, span := otel.AddSpan(ctx, "business.aclbus.delete")
	defer span.End()

	return nil
}

func (c *Core) Update(ctx context.Context, oldRule ACL, newRule ACLUpdate) error {

	ctx, span := otel.AddSpan(ctx, "business.aclbus.update")
	defer span.End()

	return nil
}

func (c *Core) GetAllPermissions(ctx context.Context) error {

	ctx, span := otel.AddSpan(ctx, "business.aclbus.getAllPermissions")
	defer span.End()

	return nil
}

func (c *Core) ValidateAccess(ctx context.Context, p ACL) error {

	ctx, span := otel.AddSpan(ctx, "business.aclbus.validateAccess")
	defer span.End()

	return nil
}
