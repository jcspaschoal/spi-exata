package dashboardapp

import (
	"net/http"

	"github.com/jcpaschoal/spi-exata/app/sdk/auth"
	"github.com/jcpaschoal/spi-exata/app/sdk/mid"
	"github.com/jcpaschoal/spi-exata/business/domain/dashboardbus"
	"github.com/jcpaschoal/spi-exata/business/sdk/web"
	"github.com/jcpaschoal/spi-exata/business/types/role"
)

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Auth         *auth.Auth
	DashboardBus *dashboardbus.Core
}

// Routes adds specific routes for this group.
func Routes(app *web.App, cfg Config) {
	const version = "v1"

	authen := mid.Authenticate(cfg.Auth)

	// Regras de negócio: Apenas ADMIN e ANALYST podem alterar o Dashboard.
	// USER pode apenas visualizar (query).
	adminOnly := mid.Authorize(cfg.Auth, role.Admin)
	canWrite := mid.Authorize(cfg.Auth, role.Admin, role.Analyst)
	canUpdate := mid.Authorize(cfg.Auth, role.Admin, role.Analyst)

	api := newApp(cfg.DashboardBus)

	app.HandlerFunc(http.MethodGet, version, "/dashboard", api.query, canUpdate)

	// POST /v1/dashboard
	app.HandlerFunc(http.MethodPost, version, "/dashboard", api.create, authen, adminOnly)

	// PUT /v1/dashboard
	app.HandlerFunc(http.MethodPut, version, "/dashboard", api.update, authen, canWrite)
}
