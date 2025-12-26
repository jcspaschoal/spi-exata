package mid

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/app/sdk/auth"
	"github.com/jcpaschoal/spi-exata/app/sdk/errs"
	"github.com/jcpaschoal/spi-exata/business/sdk/web"
)

// Authenticate valida o token JWT contido no header Authorization.
// Também realiza o "Tenant Binding", verificando se o Tenant do token
// corresponde ao Tenant da URL (se resolvido anteriormente).
func Authenticate(a *auth.Auth) web.MidFunc {
	m := func(next web.HandlerFunc) web.HandlerFunc {
		h := func(ctx context.Context, r *http.Request) web.Encoder {

			authStr := r.Header.Get("authorization")
			if authStr == "" {
				return errs.New(errs.Unauthenticated, errors.New("missing authorization header"))
			}

			parts := strings.Split(authStr, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return errs.New(errs.Unauthenticated, errors.New("expected authorization header format: Bearer <token>"))
			}

			claims, err := a.Authenticate(ctx, authStr)
			if err != nil {
				return errs.New(errs.Unauthenticated, err)
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				return errs.New(errs.Unauthenticated, fmt.Errorf("invalid user id: %w", err))
			}

			dashID, err := uuid.Parse(claims.DashboardID)
			if err != nil {
				return errs.New(errs.Unauthenticated, fmt.Errorf("invalid dashboard id: %w", err))
			}

			var tdID uuid.UUID

			if claims.TenantID != "" {
				tdID, err = uuid.Parse(claims.TenantID)
				if err != nil {
					return errs.New(errs.Unauthenticated, fmt.Errorf("invalid tenant id: %w", err))
				}
			}
			ctx = setUserID(ctx, userID)
			ctx = setTenantID(ctx, tdID)
			ctx = setDashboardID(ctx, dashID)
			ctx = setClaims(ctx, claims)

			return next(ctx, r)
		}

		return h
	}

	return m
}
