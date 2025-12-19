package userbus

import (
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/types/name"
	"github.com/jcpaschoal/spi-exata/business/types/password"
	"github.com/jcpaschoal/spi-exata/business/types/phone"
	"github.com/jcpaschoal/spi-exata/business/types/role"
)

type User struct {
	ID           uuid.UUID
	Name         name.Name
	Email        mail.Address
	Role         role.Role
	PasswordHash []byte
	Phone        phone.Null
	Enabled      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewUser contains information needed to create a new user.
type NewUser struct {
	Name     name.Name
	Email    mail.Address
	Phone    phone.Null
	Role     role.Role
	Password password.Password
}

// UpdateUser contains information needed to update a user.
type UpdateUser struct {
	Name     *name.Name
	Email    *mail.Address
	Role     *role.Role
	Phone    *phone.Null
	Password *password.Password
	Enabled  *bool
}
