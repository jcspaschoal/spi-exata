package userdb

import (
	"database/sql"
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus"
	"github.com/jcpaschoal/spi-exata/business/types/name"
	"github.com/jcpaschoal/spi-exata/business/types/phone"
	"github.com/jcpaschoal/spi-exata/business/types/role"
)

type userDB struct {
	ID           uuid.UUID      `db:"user_id"`
	Name         string         `db:"name"`
	Email        string         `db:"email"`
	Role         string         `db:"role"`
	PasswordHash []byte         `db:"password_hash"`
	Phone        sql.NullString `db:"phone"`
	Enabled      bool           `db:"enabled"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

func toDBUser(bus userbus.User) userDB {
	return userDB{
		ID:           bus.ID,
		Name:         bus.Name.String(),
		Email:        bus.Email.Address,
		Role:         bus.Role.String(),
		PasswordHash: bus.PasswordHash,
		Phone:        phone.ToSQLNullString(bus.Phone),
		Enabled:      bus.Enabled,
		CreatedAt:    bus.CreatedAt.UTC(),
		UpdatedAt:    bus.UpdatedAt.UTC(),
	}
}

func toBusUser(db userDB) (userbus.User, error) {
	addr := mail.Address{
		Address: db.Email,
	}

	usrRole, err := role.Parse(db.Role)
	if err != nil {
		return userbus.User{}, fmt.Errorf("parse: %w", err)
	}

	nme, err := name.Parse(db.Name)
	if err != nil {
		return userbus.User{}, fmt.Errorf("parse name: %w", err)
	}

	phone, err := phone.ParseNull(db.Phone.String)
	if err != nil {
		return userbus.User{}, fmt.Errorf("parse phone: %w", err)
	}

	bus := userbus.User{
		ID:           db.ID,
		Name:         nme,
		Email:        addr,
		Role:         usrRole,
		PasswordHash: db.PasswordHash,
		Enabled:      db.Enabled,
		Phone:        phone,
		CreatedAt:    db.CreatedAt.In(time.Local),
		UpdatedAt:    db.UpdatedAt.In(time.Local),
	}

	return bus, nil
}

func toBusUsers(dbs []userDB) ([]userbus.User, error) {
	bus := make([]userbus.User, len(dbs))

	for i, db := range dbs {
		var err error
		bus[i], err = toBusUser(db)
		if err != nil {
			return nil, err
		}
	}

	return bus, nil
}
