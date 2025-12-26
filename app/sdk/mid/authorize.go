package mid

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jcpaschoal/spi-exata/app/sdk/auth"
	"github.com/jcpaschoal/spi-exata/app/sdk/errs"
	"github.com/jcpaschoal/spi-exata/business/sdk/web"
	"github.com/jcpaschoal/spi-exata/business/types/role"
)

func Authorize(ath *auth.Auth, allowedRoles ...role.Role) web.MidFunc {
	m := func(next web.HandlerFunc) web.HandlerFunc {
		h := func(ctx context.Context, r *http.Request) web.Encoder {

			claims := GetClaims(ctx)
			if claims.Subject == "" {
				return errs.New(errs.Unauthenticated, errors.New("claims missing from context: authorize called without authenticate?"))
			}

			if err := ath.Authorize(ctx, claims, allowedRoles...); err != nil {
				return errs.New(errs.PermissionDenied, fmt.Errorf("authorization failed: %w", err))
			}

			return next(ctx, r)
		}

		return h
	}

	return m
}
