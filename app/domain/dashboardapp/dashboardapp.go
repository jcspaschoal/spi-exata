package dashboardapp

import (
	"context"
	"net/http"

	"github.com/jcpaschoal/spi-exata/app/sdk/errs"
	"github.com/jcpaschoal/spi-exata/app/sdk/mid"
	"github.com/jcpaschoal/spi-exata/business/domain/dashboardbus"
	"github.com/jcpaschoal/spi-exata/business/sdk/web"
)

type app struct {
	dashboardBus *dashboardbus.Core
}

func newApp(dashboardBus *dashboardbus.Core) *app {
	return &app{
		dashboardBus: dashboardBus,
	}
}

// create adds a new dashboard to the system.
func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var req NewDashboard
	if err := web.Decode(r, &req); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	nd, err := toBusNewDashboard(req)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	d, err := a.dashboardBus.Create(ctx, nd)
	if err != nil {
		return errs.Errorf(errs.Internal, "create dashboard: %s", err)
	}

	return toAppDashboard(d)
}

// query returns the dashboard details for the current user's context.
func (a *app) query(ctx context.Context, r *http.Request) web.Encoder {
	dashboardID, err := mid.GetDashboardID(ctx)
	if err != nil {
		return errs.New(errs.Unauthenticated, err)
	}

	d, err := a.dashboardBus.QueryByID(ctx, dashboardID)
	if err != nil {
		return errs.Errorf(errs.Internal, "query dashboard: %s", err)
	}

	return toAppDashboard(d)
}

// update updates the dashboard details.
func (a *app) update(ctx context.Context, r *http.Request) web.Encoder {
	var req UpdateDashboard
	if err := web.Decode(r, &req); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	dashboardID, err := mid.GetDashboardID(ctx)
	if err != nil {
		return errs.New(errs.Unauthenticated, err)
	}

	d, err := a.dashboardBus.QueryByID(ctx, dashboardID)
	if err != nil {
		return errs.Errorf(errs.Internal, "query dashboard: %s", err)
	}

	ud, err := toBusUpdateDashboard(req)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	updatedD, err := a.dashboardBus.Update(ctx, d, ud)
	if err != nil {
		return errs.Errorf(errs.Internal, "update dashboard: %s", err)
	}

	return toAppDashboard(updatedD)
}
