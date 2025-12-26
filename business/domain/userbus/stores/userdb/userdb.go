package userdb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/mail"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus"
	"github.com/jcpaschoal/spi-exata/business/sdk/order"
	"github.com/jcpaschoal/spi-exata/business/sdk/page"
	"github.com/jcpaschoal/spi-exata/business/sdk/sqldb"
	"github.com/jcpaschoal/spi-exata/foundation/logger"
	"github.com/jmoiron/sqlx"
)

// Store manages the set of APIs for user database access.
type Store struct {
	log *logger.Logger
	db  sqlx.ExtContext
}

// NewStore constructs the api for data access.
func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{
		log: log,
		db:  db,
	}
}

// NewWithTx constructs a new Store value replacing the sqlx DB
// value with a sqlx DB value that is currently inside a transaction.
func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (userbus.Storer, error) {
	ec, err := sqldb.GetExtContext(tx)
	if err != nil {
		return nil, err
	}

	store := Store{
		log: s.log,
		db:  ec,
	}

	return &store, nil
}

// Create inserts a new user into the database.
func (s *Store) Create(ctx context.Context, usr userbus.User) error {
	// Truque SQL: INSERT com SELECT
	// Pegamos o role_id da tabela 'role' baseado no nome (:role) passado no struct
	const q = `
	INSERT INTO "public"."users"
		(user_id, role_id, name, email, password, phone, enabled, created_at, updated_at)
	VALUES
		(:user_id, (SELECT role_id FROM "public"."role" WHERE name = :role), :name, :email, :password_hash, :phone, :enabled, :created_at, :updated_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBUser(usr)); err != nil {
		var dupErr sqldb.ErrDBDuplicatedEntry
		if errors.As(err, &dupErr) {
			switch dupErr.Column {
			case "email", "uq_user_email":
				return fmt.Errorf("namedexeccontext: %w", userbus.ErrUniqueEmail)
			case "phone", "uq_user_phone":
				return fmt.Errorf("namedexeccontext: %w", userbus.ErrUniquePhone)
			}
		}
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Update replaces a user document in the database.
func (s *Store) Update(ctx context.Context, usr userbus.User) error {
	// Truque SQL: Update do role_id usando subquery
	const q = `
	UPDATE
		"public"."users"
	SET 
		name = :name,
		email = :email,
		phone = :phone,
		role_id = (SELECT role_id FROM "public"."role" WHERE name = :role),
		password = :password_hash,
		enabled = :enabled,
		updated_at = :updated_at
	WHERE
		user_id = :user_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBUser(usr)); err != nil {
		var dupErr sqldb.ErrDBDuplicatedEntry
		if errors.As(err, &dupErr) {
			switch dupErr.Column {
			case "email", "uq_user_email":
				return userbus.ErrUniqueEmail
			case "phone", "uq_user_phone":
				return userbus.ErrUniquePhone
			}
		}
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Delete removes a user from the database.
func (s *Store) Delete(ctx context.Context, usr userbus.User) error {
	const q = `
	DELETE FROM
		"public"."users"
	WHERE
		user_id = :user_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBUser(usr)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Query retrieves a list of existing users from the database.
func (s *Store) Query(ctx context.Context, filter userbus.QueryFilter, orderBy order.By, page page.Page) ([]userbus.User, error) {
	data := map[string]any{
		"offset":        (page.Number() - 1) * page.RowsPerPage(),
		"rows_per_page": page.RowsPerPage(),
	}

	// Fazemos JOIN com a tabela role para pegar o nome da role (r.name)
	// e mapear para o campo "role" do struct userDB.
	// Alias 'password_hash' é necessário pois no banco é 'password' mas no struct é 'password_hash'
	const q = `
	SELECT
		u.user_id, u.name, u.email, u.password AS password_hash, u.phone, u.enabled, u.created_at, u.updated_at,
		r.name AS role
	FROM
		"public"."users" AS u
	JOIN
		"public"."role" AS r ON r.role_id = u.role_id`

	buf := bytes.NewBufferString(q)
	applyFilter(filter, data, buf)

	orderByClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(orderByClause)
	buf.WriteString(" OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY")

	var dbUsrs []userDB
	if err := sqldb.NamedQuerySlice(ctx, s.log, s.db, buf.String(), data, &dbUsrs); err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusUsers(dbUsrs)
}

// Count returns the total number of users in the DB.
func (s *Store) Count(ctx context.Context, filter userbus.QueryFilter) (int, error) {
	data := map[string]any{}

	const q = `
	SELECT
		count(1)
	FROM
		"public"."users" AS u` // Alias 'u' caso o filtro use prefixo u.

	buf := bytes.NewBufferString(q)
	applyFilter(filter, data, buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("db: %w", err)
	}

	return count.Count, nil
}

// QueryByID gets the specified user from the database.
func (s *Store) QueryByID(ctx context.Context, userID uuid.UUID) (userbus.User, error) {
	data := struct {
		ID string `db:"user_id"`
	}{
		ID: userID.String(),
	}

	const q = `
	SELECT
		u.user_id, u.name, u.email, u.password AS password_hash, u.phone, u.enabled, u.created_at, u.updated_at,
		r.name AS role
	FROM
		"public"."users" AS u
	JOIN
		"public"."role" AS r ON r.role_id = u.role_id
	WHERE 
		u.user_id = :user_id`

	var dbUsr userDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &dbUsr); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return userbus.User{}, fmt.Errorf("db: %w", userbus.ErrNotFound)
		}
		return userbus.User{}, fmt.Errorf("db: %w", err)
	}

	return toBusUser(dbUsr)
}

// QueryByEmail gets the specified user from the database by email.
func (s *Store) QueryByEmail(ctx context.Context, email mail.Address) (userbus.User, error) {
	data := struct {
		Email string `db:"email"`
	}{
		Email: email.Address,
	}

	const q = `
	SELECT
		u.user_id, u.name, u.email, u.password AS password_hash, u.phone, u.enabled, u.created_at, u.updated_at,
		r.name AS role
	FROM
		"public"."users" AS u
	JOIN
		"public"."role" AS r ON r.role_id = u.role_id
	WHERE
		u.email = :email`

	var dbUsr userDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &dbUsr); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return userbus.User{}, fmt.Errorf("db: %w", userbus.ErrNotFound)
		}
		return userbus.User{}, fmt.Errorf("db: %w", err)
	}

	return toBusUser(dbUsr)
}
