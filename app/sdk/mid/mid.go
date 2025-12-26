package mid

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/app/sdk/auth"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus"
	"github.com/jcpaschoal/spi-exata/business/sdk/sqldb"
	"github.com/jcpaschoal/spi-exata/business/sdk/web"
)

func checkIsError(e web.Encoder) error {
	err, hasError := e.(error)
	if hasError {
		return err
	}

	return nil
}

// =============================================================================

type ctxKey int

const (
	claimKey ctxKey = iota + 1
	userIDKey
	userKey
	trKey
	keyTenantID
	dashboardID
)

func setTenantID(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, keyTenantID, tenantID)
}

func GetTenantID(ctx context.Context) (uuid.UUID, error) {
	v, ok := ctx.Value(keyTenantID).(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("tenant id not found in context")
	}
	return v, nil
}

func setDashboardID(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, keyTenantID, tenantID)
}

func GetDashboardID(ctx context.Context) (uuid.UUID, error) {
	v, ok := ctx.Value(keyTenantID).(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("tenant id not found in context")
	}
	return v, nil
}

func setClaims(ctx context.Context, claims auth.Claims) context.Context {
	return context.WithValue(ctx, claimKey, claims)
}

// GetClaims returns the claims from the context.
func GetClaims(ctx context.Context) auth.Claims {
	v, ok := ctx.Value(claimKey).(auth.Claims)
	if !ok {
		return auth.Claims{}
	}
	return v
}

// GetSubjectID returns the subject id from the claims.
func GetSubjectID(ctx context.Context) uuid.UUID {
	v := GetClaims(ctx)

	subjectID, err := uuid.Parse(v.Subject)
	if err != nil {
		return uuid.UUID{}
	}

	return subjectID
}

func setUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID returns the user id from the context.
func GetUserID(ctx context.Context) (uuid.UUID, error) {
	v, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		return uuid.UUID{}, errors.New("user id not found in context")
	}

	return v, nil
}

func setUser(ctx context.Context, usr userbus.User) context.Context {
	return context.WithValue(ctx, userKey, usr)
}

// GetUser returns the user from the context.
func GetUser(ctx context.Context) (userbus.User, error) {
	v, ok := ctx.Value(userKey).(userbus.User)
	if !ok {
		return userbus.User{}, errors.New("user not found in context")
	}

	return v, nil
}

func setTran(ctx context.Context, tx sqldb.CommitRollbacker) context.Context {
	return context.WithValue(ctx, trKey, tx)
}

// GetTran retrieves the value that can manage a transaction.
func GetTran(ctx context.Context) (sqldb.CommitRollbacker, error) {
	v, ok := ctx.Value(trKey).(sqldb.CommitRollbacker)
	if !ok {
		return nil, errors.New("transaction not found in context")
	}

	return v, nil
}
