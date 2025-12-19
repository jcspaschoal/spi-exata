package mux

import (
	"embed"
	"net/http"

	"github.com/jcpaschoal/spi-exata/business/domain/userbus"
	"github.com/jcpaschoal/spi-exata/business/sdk/web"
	"github.com/jcpaschoal/spi-exata/foundatiton/logger"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/trace"
)

type StaticSite struct {
	react      bool
	static     embed.FS
	staticDir  string
	staticPath string
}

// Options represent optional parameters.
type Options struct {
	corsOrigin []string
	sites      []StaticSite
}

// WithCORS provides configuration options for CORS.
func WithCORS(origins []string) func(opts *Options) {
	return func(opts *Options) {
		opts.corsOrigin = origins
	}
}

// WithFileServer provides configuration options for file server.
func WithFileServer(react bool, static embed.FS, dir string, path string) func(opts *Options) {
	return func(opts *Options) {
		opts.sites = append(opts.sites, StaticSite{
			react:      react,
			static:     static,
			staticDir:  dir,
			staticPath: path,
		})
	}
}

// SalesConfig contains sales service specific config.
type SalesConfig struct {
	AuthClient authclient.Authenticator
}

// AuthConfig contains auth service specific config.
type AuthConfig struct {
	Auth *auth.Auth
}

type BusConfig struct {
	UserBus userbus.Core
}

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Build       string
	Log         *logger.Logger
	DB          *sqlx.DB
	Tracer      trace.Tracer
	BusConfig   BusConfig
	SalesConfig SalesConfig
	AuthConfig  AuthConfig
}

// RouteAdder defines behavior that sets the routes to bind for an instance
// of the service.
type RouteAdder interface {
	Add(app *web.App, cfg Config)
}

// WebAPI constructs a http.Handler with all application routes bound.
func WebAPI(cfg Config, routeAdder RouteAdder, options ...func(opts *Options)) http.Handler {
	app := web.NewApp(
		cfg.Log.Info,
		cfg.Tracer,
		mid.Otel(cfg.Tracer),
		mid.Logger(cfg.Log),
		mid.Errors(cfg.Log),
		mid.Metrics(),
		mid.Panics(),
	)

	var opts Options
	for _, option := range options {
		option(&opts)
	}

	if len(opts.corsOrigin) > 0 {
		app.EnableCORS(opts.corsOrigin)
	}

	routeAdder.Add(app, cfg)

	for _, site := range opts.sites {
		switch site.react {
		case true:
			app.FileServerReact(site.static, site.staticDir, site.staticPath)

		default:
			app.FileServer(site.static, site.staticDir, site.staticPath)
		}
	}

	return app
}
