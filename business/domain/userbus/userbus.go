package userbus

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/sdk/order"
	"github.com/jcpaschoal/spi-exata/business/sdk/page"
	"github.com/jcpaschoal/spi-exata/business/sdk/sqldb"
	"github.com/jcpaschoal/spi-exata/foundatiton/otel"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotFound              = errors.New("user not found")
	ErrUniqueEmail           = errors.New("email is not unique")
	ErrUniquePhone           = errors.New("Phone is not unique")
	ErrAuthenticationFailure = errors.New("authentication failed")
)

type Storer interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Storer, error)
	Create(ctx context.Context, usr User) error
	Update(ctx context.Context, usr User) error
	Delete(ctx context.Context, usr User) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]User, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, userID uuid.UUID) (User, error)
	QueryByEmail(ctx context.Context, email mail.Address) (User, error)
}

type Core struct {
	storer Storer
}

func NewCore(storer Storer) *Core {
	return &Core{
		storer: storer,
	}
}

func (c *Core) NewWithTx(tx sqldb.CommitRollbacker) (*Core, error) {
	storer, err := c.storer.NewWithTx(tx)

	if err != nil {
		return nil, err
	}

	nc := NewCore(storer)

	return nc, nil

}

func (c *Core) Create(ctx context.Context, nu NewUser) (User, error) {

	ctx, span := otel.AddSpan(ctx, "business.userbus.create")
	defer span.End()

	hash, err := bcrypt.GenerateFromPassword([]byte(nu.Password.String()), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("generateFromPassword: %w", err)
	}

	now := time.Now()

	usr := User{
		ID:           uuid.New(),
		Name:         nu.Name,
		Email:        nu.Email,
		PasswordHash: hash,
		Role:         nu.Role,
		Phone:        nu.Phone,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := c.storer.Create(ctx, usr); err != nil {
		return User{}, fmt.Errorf("create: %w", err)
	}

	return usr, nil
}

func (c *Core) Update(ctx context.Context, usr User, uu UpdateUser) (User, error) {

	ctx, span := otel.AddSpan(ctx, "business.userbus.update")
	defer span.End()

	if uu.Name != nil {
		usr.Name = *uu.Name
	}

	if uu.Email != nil {
		usr.Email = *uu.Email
	}

	if uu.Role != nil {
		usr.Role = *uu.Role
	}

	if uu.Password != nil {
		pw, err := bcrypt.GenerateFromPassword([]byte(uu.Password.String()), bcrypt.DefaultCost)
		if err != nil {
			return User{}, fmt.Errorf("generatefrompassword: %w", err)
		}
		usr.PasswordHash = pw
	}

	if uu.Phone != nil {
		usr.Phone = *uu.Phone
	}

	if uu.Enabled != nil {
		usr.Enabled = *uu.Enabled
	}

	usr.UpdatedAt = time.Now()

	if err := c.storer.Update(ctx, usr); err != nil {
		return User{}, fmt.Errorf("update: %w", err)
	}

	return usr, nil
}

func (c *Core) Delete(ctx context.Context, usr User) error {

	ctx, span := otel.AddSpan(ctx, "business.userbus.delete")
	defer span.End()

	if err := c.storer.Delete(ctx, usr); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

// Query retrieves a list of existing users.
func (c *Core) Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]User, error) {

	ctx, span := otel.AddSpan(ctx, "business.userbus.query")
	defer span.End()

	users, err := c.storer.Query(ctx, filter, orderBy, page)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	return users, nil
}

// Count returns the total number of users.
func (c *Core) Count(ctx context.Context, filter QueryFilter) (int, error) {
	ctx, span := otel.AddSpan(ctx, "business.userbus.count")
	defer span.End()

	return c.storer.Count(ctx, filter)
}

// QueryByID finds the user by the specified ID.
func (c *Core) QueryByID(ctx context.Context, userID uuid.UUID) (User, error) {

	ctx, span := otel.AddSpan(ctx, "business.userbus.queryByID")
	defer span.End()

	user, err := c.storer.QueryByID(ctx, userID)
	if err != nil {
		return User{}, fmt.Errorf("query: userID[%s]: %w", userID, err)
	}

	return user, nil
}

// QueryByEmail finds the user by a specified user email.
func (c *Core) QueryByEmail(ctx context.Context, email mail.Address) (User, error) {

	ctx, span := otel.AddSpan(ctx, "business.userbus.queryByEmail")
	defer span.End()

	user, err := c.storer.QueryByEmail(ctx, email)
	if err != nil {
		return User{}, fmt.Errorf("query: email[%s]: %w", email, err)
	}

	return user, nil
}

// Authenticate finds a user by their email and verifies their password. On
// success it returns a Claims User representing this user. The claims can be
// used to generate a token for future authentication.
func (c *Core) Authenticate(ctx context.Context, email mail.Address, password string) (User, error) {

	ctx, span := otel.AddSpan(ctx, "business.userbus.authenticate")
	defer span.End()

	usr, err := c.QueryByEmail(ctx, email)
	if err != nil {
		return User{}, fmt.Errorf("query: email[%s]: %w", email, err)
	}

	if err := bcrypt.CompareHashAndPassword(usr.PasswordHash, []byte(password)); err != nil {
		return User{}, fmt.Errorf("compareHashAndPassword: %w", ErrAuthenticationFailure)
	}

	return usr, nil
}
