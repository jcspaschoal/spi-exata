package userapp

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"time"

	"github.com/jcpaschoal/spi-exata/app/sdk/errs"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus"
	"github.com/jcpaschoal/spi-exata/business/types/name"
	"github.com/jcpaschoal/spi-exata/business/types/password"
	"github.com/jcpaschoal/spi-exata/business/types/phone"
	"github.com/jcpaschoal/spi-exata/business/types/role"
)

// =============================================================================
// User (Output)
// =============================================================================

// User represents information about an individual user.
type User struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	Phone       string `json:"phone"`
	Enabled     bool   `json:"enabled"`
	DateCreated string `json:"dateCreated"`
	DateUpdated string `json:"dateUpdated"`
}

// Encode implements the web.Encoder interface.
func (u User) Encode() ([]byte, string, error) {
	data, err := json.Marshal(u)
	return data, "application/json", err
}

func toAppUser(bus userbus.User) User {
	return User{
		ID:          bus.ID.String(),
		Name:        bus.Name.String(),
		Email:       bus.Email.Address,
		Role:        bus.Role.String(),
		Phone:       bus.Phone.String(),
		Enabled:     bus.Enabled,
		DateCreated: bus.CreatedAt.Format(time.RFC3339),
		DateUpdated: bus.UpdatedAt.Format(time.RFC3339),
	}
}

func toAppUsers(users []userbus.User) []User {
	app := make([]User, len(users))
	for i, usr := range users {
		app[i] = toAppUser(usr)
	}
	return app
}

// =============================================================================
// NewUser (Input)
// =============================================================================

// NewUser defines the data needed to add a new user.
type NewUser struct {
	Name            string `json:"name" validate:"required"`
	Email           string `json:"email" validate:"required,email"`
	Role            string `json:"role" validate:"required"`
	Phone           string `json:"phone"`
	Password        string `json:"password" validate:"required"`
	PasswordConfirm string `json:"passwordConfirm" validate:"eqfield=Password"`
}

// Decode implements the web.Decoder interface.
func (app *NewUser) Decode(data []byte) error {
	return json.Unmarshal(data, app)
}

// Validate checks the data in the model is considered clean.
func (app NewUser) Validate() error {
	if err := errs.Check(app); err != nil {
		return errs.New(errs.InvalidArgument, fmt.Errorf("validate: %w", err))
	}
	return nil
}

func toBusNewUser(app NewUser) (userbus.NewUser, error) {
	parsedRole, err := role.Parse(app.Role)
	if err != nil {
		return userbus.NewUser{}, fmt.Errorf("parse role: %w", err)
	}

	addr, err := mail.ParseAddress(app.Email)
	if err != nil {
		return userbus.NewUser{}, fmt.Errorf("parse email: %w", err)
	}

	nme, err := name.Parse(app.Name)
	if err != nil {
		return userbus.NewUser{}, fmt.Errorf("parse name: %w", err)
	}

	ph, err := phone.ParseNull(app.Phone)
	if err != nil {
		return userbus.NewUser{}, fmt.Errorf("parse phone: %w", err)
	}

	pass, err := password.Parse(app.Password)
	if err != nil {
		return userbus.NewUser{}, fmt.Errorf("parse password: %w", err)
	}

	bus := userbus.NewUser{
		Name:     nme,
		Email:    *addr,
		Role:     parsedRole,
		Phone:    ph,
		Password: pass,
	}

	return bus, nil
}

// =============================================================================
// UpdateUserRole (Input)
// =============================================================================

// UpdateUserRole defines the data needed to update a user role.
type UpdateUserRole struct {
	Role string `json:"role" validate:"required"`
}

// Decode implements the web.Decoder interface.
func (app *UpdateUserRole) Decode(data []byte) error {
	return json.Unmarshal(data, app)
}

// Validate checks the data in the model is considered clean.
func (app UpdateUserRole) Validate() error {
	if err := errs.Check(app); err != nil {
		return errs.New(errs.InvalidArgument, fmt.Errorf("validate: %w", err))
	}
	return nil
}

func toBusUpdateUserRole(app UpdateUserRole) (userbus.UpdateUser, error) {
	r, err := role.Parse(app.Role)
	if err != nil {
		return userbus.UpdateUser{}, fmt.Errorf("parse role: %w", err)
	}

	bus := userbus.UpdateUser{
		Role: &r,
	}

	return bus, nil
}

// =============================================================================
// UpdateUser (Input)
// =============================================================================

// UpdateUser defines the data needed to update a user.
type UpdateUser struct {
	Name            *string `json:"name"`
	Email           *string `json:"email" validate:"omitempty,email"`
	Phone           *string `json:"phone"`
	Password        *string `json:"password"`
	PasswordConfirm *string `json:"passwordConfirm" validate:"omitempty,eqfield=Password"`
	Enabled         *bool   `json:"enabled"`
}

// Decode implements the web.Decoder interface.
func (app *UpdateUser) Decode(data []byte) error {
	return json.Unmarshal(data, app)
}

// Validate checks the data in the model is considered clean.
func (app UpdateUser) Validate() error {
	if err := errs.Check(app); err != nil {
		return errs.New(errs.InvalidArgument, fmt.Errorf("validate: %w", err))
	}
	return nil
}

func toBusUpdateUser(app UpdateUser) (userbus.UpdateUser, error) {
	var addr *mail.Address
	if app.Email != nil {
		var err error
		addr, err = mail.ParseAddress(*app.Email)
		if err != nil {
			return userbus.UpdateUser{}, fmt.Errorf("parse email: %w", err)
		}
	}

	var nme *name.Name
	if app.Name != nil {
		nm, err := name.Parse(*app.Name)
		if err != nil {
			return userbus.UpdateUser{}, fmt.Errorf("parse name: %w", err)
		}
		nme = &nm
	}

	var ph *phone.Null
	if app.Phone != nil {
		p, err := phone.ParseNull(*app.Phone)
		if err != nil {
			return userbus.UpdateUser{}, fmt.Errorf("parse phone: %w", err)
		}
		ph = &p
	}

	var pass *password.Password
	if app.Password != nil {
		p, err := password.Parse(*app.Password)
		if err != nil {
			return userbus.UpdateUser{}, fmt.Errorf("parse password: %w", err)
		}
		pass = &p
	}

	bus := userbus.UpdateUser{
		Name:     nme,
		Email:    addr,
		Phone:    ph,
		Password: pass,
		Enabled:  app.Enabled,
	}

	return bus, nil
}
