package userbus

import (
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/types/name"
)

type QueryFilter struct {
	ID             *uuid.UUID
	Name           *name.Name
	Email          *mail.Address
	StartCreatedAt *time.Time
	EndCreatedAt   *time.Time
}
