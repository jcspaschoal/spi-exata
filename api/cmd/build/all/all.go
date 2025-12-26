package all

import (
	"time"

	"github.com/jcpaschoal/spi-exata/app/domain/authapp"
	"github.com/jcpaschoal/spi-exata/app/domain/userapp"
	"github.com/jcpaschoal/spi-exata/app/sdk/auth"
	"github.com/jcpaschoal/spi-exata/app/sdk/mux"
	"github.com/jcpaschoal/spi-exata/business/domain/tenantbus"
	tenantdb "github.com/jcpaschoal/spi-exata/business/domain/tenantbus/stores"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus/stores/usercache"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus/stores/userdb"
	"github.com/jcpaschoal/spi-exata/business/sdk/web"
)

// Routes constructs the add value which provides the implementation of
// of RouteAdder for specifying what routes to bind to this instance.
func Routes() add {
	return add{}
}

type add struct{}

func (add) Add(app *web.App, cfg mux.Config) {

	userBus := userbus.NewCore(usercache.NewStore(cfg.Log, userdb.NewStore(cfg.Log, cfg.DB), time.Minute*5))
	tenantBus := tenantbus.NewCore(cfg.Log, tenantdb.NewStore(cfg.Log, cfg.DB))

	authClient := auth.New(auth.Config{
		Log:       cfg.Log,
		UserBus:   userBus,
		KeyLookup: cfg.AuthConfig.KeyLookup,
		Issuer:    cfg.AuthConfig.Issuer,
		ActiveKID: cfg.AuthConfig.ActiveKID,
	})

	userapp.Routes(app, userapp.Config{
		Auth:    authClient,
		UserBus: userBus,
	})

	authapp.Routes(app, authapp.Config{
		Auth:      authClient,
		TenantBus: tenantBus,
	})

}
