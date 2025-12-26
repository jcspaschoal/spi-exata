package authapp

import (
	"context"
	"fmt"
	"net/http"
	"net/mail"

	"github.com/jcpaschoal/spi-exata/app/sdk/auth"
	"github.com/jcpaschoal/spi-exata/app/sdk/errs"
	"github.com/jcpaschoal/spi-exata/business/domain/tenantbus"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus"
	"github.com/jcpaschoal/spi-exata/business/sdk/web"
	"github.com/jcpaschoal/spi-exata/business/types/role"
)

type app struct {
	auth      *auth.Auth
	tenantBus *tenantbus.Core
}

// newApp constructs a user app API for use.
func newApp(auth *auth.Auth, tenantBus *tenantbus.Core, userBus *userbus.Core) *app {
	return &app{
		auth:      auth,
		tenantBus: tenantBus,
	}
}

func (a *app) login(ctx context.Context, r *http.Request) web.Encoder {
	var req Login

	if err := web.Decode(r, &req); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	addr, err := mail.ParseAddress(req.Email)
	if err != nil {
		return errs.New(errs.InvalidArgument, fmt.Errorf("parsing email: %w", err))
	}

	usr, err := a.auth.Login(ctx, *addr, req.Password)
	if err != nil {
		return errs.New(errs.Unauthenticated, err)
	}

	domain := auth.ExtractDomain(r.Host)

	var td tenantbus.TenantDashboard

	if usr.Role.Equal(role.User) {
		td, err = a.tenantBus.AuthorizeUserAccessToDashboard(ctx, usr.ID, domain)
		if err != nil {
			return errs.New(errs.PermissionDenied, tenantbus.ErrAccessDenied)
		}
	}

	tokenStr, err := a.auth.GenerateToken(usr.ID.String(), td.TenantID, usr.ID, td.DashboardID, usr.Role)
	if err != nil {
		return errs.New(errs.Internal, err)
	}

	return toAppToken(tokenStr)
}
