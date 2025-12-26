package userapp

import (
	"net/http"

	"github.com/jcpaschoal/spi-exata/app/sdk/auth"
	"github.com/jcpaschoal/spi-exata/app/sdk/mid"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus"
	"github.com/jcpaschoal/spi-exata/business/sdk/web"
	"github.com/jcpaschoal/spi-exata/business/types/role"
)

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Auth    *auth.Auth
	UserBus *userbus.Core
}

// Routes adds specific routes for this group.
func Routes(app *web.App, cfg Config) {
	const version = "v1"

	// Middlewares
	authen := mid.Authenticate(cfg.Auth)

	// Instanciamos a API
	api := newApp(cfg.Auth, cfg.UserBus)

	// GET /users
	app.HandlerFunc(http.MethodGet, version, "/users", api.query, authen, mid.Authorize(cfg.Auth, role.Admin))

	// GET /users/{user_id}
	app.HandlerFunc(http.MethodGet, version, "/users/{user_id}", api.queryByID, authen, mid.Authorize(cfg.Auth, role.Admin))

	// POST /users
	app.HandlerFunc(http.MethodPost, version, "/users", api.create, authen, mid.Authorize(cfg.Auth, role.Admin))

	// PUT /users/{user_id}
	app.HandlerFunc(http.MethodPut, version, "/me", api.update, authen)

	// DELETE /users/{user_id}
	app.HandlerFunc(http.MethodDelete, version, "/me", api.delete, authen)
}
