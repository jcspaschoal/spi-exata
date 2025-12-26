package userapp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/mail"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/app/sdk/auth"
	"github.com/jcpaschoal/spi-exata/app/sdk/errs"
	"github.com/jcpaschoal/spi-exata/app/sdk/mid"
	"github.com/jcpaschoal/spi-exata/app/sdk/query"
	"github.com/jcpaschoal/spi-exata/business/domain/tenantbus"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus"
	"github.com/jcpaschoal/spi-exata/business/sdk/order"
	"github.com/jcpaschoal/spi-exata/business/sdk/page"
	"github.com/jcpaschoal/spi-exata/business/sdk/web"
	"github.com/jcpaschoal/spi-exata/business/types/role"
)

// app manages the set of app layer api functions for the user domain.
// Nota: Até a struct pode ser privada (app) se só for usada aqui e no route.go
type app struct {
	auth      *auth.Auth
	tenantBus *tenantbus.Core
	userBus   *userbus.Core
}

// newApp constructs a user app API for use.
func newApp(auth *auth.Auth, tenantBus *tenantbus.Core, userBus *userbus.Core) *app {
	return &app{
		auth:      auth,
		tenantBus: tenantBus,
		userBus:   userBus,
	}
}

// create adds a new user to the system.
func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var app NewUser
	if err := web.Decode(r, &app); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	nc, err := toBusNewUser(app)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	usr, err := a.userBus.Create(ctx, nc)
	if err != nil {
		if errors.Is(err, userbus.ErrUniqueEmail) {
			return errs.New(errs.Aborted, userbus.ErrUniqueEmail)
		}
		if errors.Is(err, userbus.ErrUniquePhone) {
			return errs.New(errs.Aborted, userbus.ErrUniquePhone)
		}
		return errs.Errorf(errs.Internal, "create: usr[%+v]: %s", usr, err)
	}

	return &CreatedUser{User: toAppUser(usr)}
}

// update updates an existing user.
func (a *app) update(ctx context.Context, r *http.Request) web.Encoder {
	var app UpdateUser
	if err := web.Decode(r, &app); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	usr, err := mid.GetUser(ctx)
	if err != nil {
		return errs.Errorf(errs.Internal, "user missing in context: %s", err)
	}

	uu, err := toBusUpdateUser(app)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	updUsr, err := a.userBus.Update(ctx, usr, uu)
	if err != nil {
		return errs.Errorf(errs.Internal, "update: userID[%s] uu[%+v]: %s", usr.ID, uu, err)
	}

	return toAppUser(updUsr)
}

// updateRole updates an existing user's role.
func (a *app) updateRole(ctx context.Context, r *http.Request) web.Encoder {
	var app UpdateUserRole
	if err := web.Decode(r, &app); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	id := r.PathValue("user_id")
	userID, err := uuid.Parse(id)
	if err != nil {
		return errs.NewFieldErrors("user_id", err)
	}

	usr, err := a.userBus.QueryByID(ctx, userID)
	if err != nil {
		if errors.Is(err, userbus.ErrNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Errorf(errs.Internal, "query user: %s", err)
	}

	uu, err := toBusUpdateUserRole(app)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	updUsr, err := a.userBus.Update(ctx, usr, uu)
	if err != nil {
		return errs.Errorf(errs.Internal, "updaterole: userID[%s] uu[%+v]: %s", usr.ID, uu, err)
	}

	return toAppUser(updUsr)
}

// delete removes a user from the system.
func (a *app) delete(ctx context.Context, _ *http.Request) web.Encoder {
	usr, err := mid.GetUser(ctx)

	if err != nil {
		return errs.Errorf(errs.Internal, "userID missing in context: %s", err)
	}

	if err := a.userBus.Delete(ctx, usr); err != nil {
		return errs.Errorf(errs.Internal, "delete: userID[%s]: %s", usr.ID, err)
	}

	return nil
}

// query returns a list of users with paging.
func (a *app) query(ctx context.Context, r *http.Request) web.Encoder {
	qp := parseQueryParams(r)

	page, err := page.Parse(qp.Page, qp.Rows)
	if err != nil {
		return errs.NewFieldErrors("page", err)
	}

	filter, err := parseFilter(qp)
	if err != nil {
		if v, ok := err.(*errs.Error); ok {
			return v
		}
		return errs.NewFieldErrors("filter", err)
	}

	orderBy, err := order.Parse(orderByFields, qp.OrderBy, userbus.DefaultOrderBy)
	if err != nil {
		return errs.NewFieldErrors("order", err)
	}

	usrs, err := a.userBus.Query(ctx, filter, orderBy, page)
	if err != nil {
		return errs.Errorf(errs.Internal, "query: %s", err)
	}

	total, err := a.userBus.Count(ctx, filter)
	if err != nil {
		return errs.Errorf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(toAppUsers(usrs), total, page)
}

// queryByID returns a user by its ID.
func (a *app) queryByID(ctx context.Context, _ *http.Request) web.Encoder {
	usr, err := mid.GetUser(ctx)
	if err != nil {
		return errs.Errorf(errs.Internal, "querybyid: %s", err)
	}

	return toAppUser(usr)
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

	usr, err := a.userBus.Authenticate(ctx, *addr, req.Password)
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
