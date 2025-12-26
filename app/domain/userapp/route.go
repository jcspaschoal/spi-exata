package userapp

import (
	"net/http"

	"github.com/jcpaschoal/spi-exata/app/sdk/auth"
	"github.com/jcpaschoal/spi-exata/app/sdk/mid"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus"
	"github.com/jcpaschoal/spi-exata/business/sdk/web"
	"github.com/jcpaschoal/spi-exata/business/types/actions"
	"github.com/jcpaschoal/spi-exata/business/types/resource"
	"github.com/jcpaschoal/spi-exata/foundatiton/logger"
)

// Vou reformular o modulo de acl para funcionar com roles , e criar uma tabela de clientes para associar aos clientes

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Log     *logger.Logger
	UserBus *userbus.Core
	Auth    *auth.Auth
}

// Routes adds specific routes for this group.
func Routes(app *web.App, cfg Config) {
	const version = "v1"

	// 1. Autenticação (Quem é você?)
	authen := mid.Authenticate(cfg.Auth)

	// Instanciamos a API (Handlers)
	api := newApp(cfg.UserBus)

	// 2. Definição de Rotas com Autorização (O que você pode fazer?)

	// GET /users -> Requer permissão para LER (Read) o recurso USUÁRIOS (User).
	app.HandlerFunc(http.MethodGet, version, "/users", api.query, authen,
		mid.Authorize(cfg.Auth, resource.User, actions.Read))

	// GET /users/{user_id} -> Requer permissão para LER a INSTÂNCIA específica (ID na URL).
	app.HandlerFunc(http.MethodGet, version, "/users/{user_id}", api.queryByID, authen,
		mid.Authorize(cfg.Auth, resource.User, actions.Read, "user_id"))

	// POST /users -> Requer permissão para CRIAR (Create) novos USUÁRIOS.
	app.HandlerFunc(http.MethodPost, version, "/users", api.create, authen,
		mid.Authorize(cfg.Auth, resource.User, actions.Create))

	// PUT /users/role/{user_id} -> Atualizar Role é uma ação sensível, tratada como UPDATE na instância.
	// (Poderíamos ter uma action específica 'Promote' se o business exigisse, mas Update cobre).
	app.HandlerFunc(http.MethodPut, version, "/users/role/{user_id}", api.updateRole, authen,
		mid.Authorize(cfg.Auth, resource.User, actions.Update, "user_id"))

	// PUT /users/{user_id} -> Requer permissão para ATUALIZAR a INSTÂNCIA específica.
	app.HandlerFunc(http.MethodPut, version, "/users/{user_id}", api.update, authen,
		mid.Authorize(cfg.Auth, resource.User, actions.Update, "user_id"))

	// DELETE /users/{user_id} -> Requer permissão para DELETAR a INSTÂNCIA específica.
	app.HandlerFunc(http.MethodDelete, version, "/users/{user_id}", api.delete, authen,
		mid.Authorize(cfg.Auth, resource.User, actions.Delete, "user_id"))
}
