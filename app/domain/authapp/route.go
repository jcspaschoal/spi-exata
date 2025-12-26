package authapp

import (
	"net/http"

	"github.com/jcpaschoal/spi-exata/app/sdk/auth"
	"github.com/jcpaschoal/spi-exata/business/domain/tenantbus"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus"
	"github.com/jcpaschoal/spi-exata/business/sdk/web"
)

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Auth      *auth.Auth
	UserBus   *userbus.Core
	TenantBus *tenantbus.Core
}

// Routes adds specific routes for this group.
func Routes(app *web.App, cfg Config) {
	const version = "v1"

	// Middlewares
	//authen := mid.Authenticate(cfg.Auth)

	// Instanciamos a API
	api := newApp(cfg.Auth, cfg.TenantBus, cfg.UserBus)

	app.HandlerFunc(http.MethodPost, version, "/auth/login", api.login)

}
