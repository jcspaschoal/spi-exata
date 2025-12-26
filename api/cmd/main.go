package main

import (
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/jcpaschoal/spi-exata/api/cmd/build/all"
	"github.com/jcpaschoal/spi-exata/app/sdk/mux"
	"github.com/jcpaschoal/spi-exata/business/sdk/sqldb"
	"github.com/jcpaschoal/spi-exata/foundation/keystore"
	"github.com/jcpaschoal/spi-exata/foundation/logger"
	"github.com/jcpaschoal/spi-exata/foundation/otel"
	"github.com/kelseyhightower/envconfig"
)

var build = "develop"
var routes = "all" // go build -ldflags "-X main.routes=crud"

// Assumindo que 'static' é definido em outro lugar ou via embed,
// caso contrário precisaria de: var static embed.FS

type Config struct {
	Version struct {
		Build string `json:"build"`
		Desc  string `json:"desc"`
	} `json:"version"`

	Web struct {
		ReadTimeout        time.Duration `envconfig:"WEB_READ_TIMEOUT" default:"5s"`
		WriteTimeout       time.Duration `envconfig:"WEB_WRITE_TIMEOUT" default:"10s"`
		IdleTimeout        time.Duration `envconfig:"WEB_IDLE_TIMEOUT" default:"120s"`
		ShutdownTimeout    time.Duration `envconfig:"WEB_SHUTDOWN_TIMEOUT" default:"20s"`
		APIHost            string        `envconfig:"WEB_API_HOST" default:"0.0.0.0:3000"`
		DebugHost          string        `envconfig:"WEB_DEBUG_HOST" default:"0.0.0.0:3010"`
		CORSAllowedOrigins []string      `envconfig:"WEB_CORS_ALLOWED_ORIGINS" default:"*"`
	}
	DB struct {
		User         string `envconfig:"DB_USER" default:"postgres"`
		Password     string `envconfig:"DB_PASSWORD" default:"postgres"`
		Host         string `envconfig:"DB_HOST" default:"localhost"`
		Name         string `envconfig:"DB_NAME" default:"spi"`
		MaxIdleConns int    `envconfig:"DB_MAX_IDLE_CONNS" default:"0"`
		MaxOpenConns int    `envconfig:"DB_MAX_OPEN_CONNS" default:"0"`
		DisableTLS   bool   `envconfig:"DB_DISABLE_TLS" default:"true"`
	}
	Tempo struct {
		Host        string  `envconfig:"TEMPO_HOST" default:"tempo:4317"`
		ServiceName string  `envconfig:"TEMPO_SERVICE_NAME" default:"SPI-EXATA"`
		Probability float64 `envconfig:"TEMPO_PROBABILITY" default:"0.05"`
		Enabled     bool    `envconfig:"TEMPO_ENABLED" default:"true"`
	}
}

func main() {
	var log *logger.Logger

	events := logger.Events{
		Error: func(ctx context.Context, r logger.Record) {
			log.Info(ctx, "******* SEND ALERT *******")
		},
	}

	log = logger.NewWithEvents(os.Stdout, logger.LevelInfo, "SPI-EXATA", otel.GetTraceID, events)

	// -------------------------------------------------------------------------

	ctx := context.Background()

	if err := run(ctx, log); err != nil {
		log.Error(ctx, "startup", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *logger.Logger) error {

	// -------------------------------------------------------------------------
	// GOMAXPROCS

	log.Info(ctx, "startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	// -------------------------------------------------------------------------
	// Configuration
	var cfg Config

	cfg.Version.Build = build
	cfg.Version.Desc = "SPI-EXATA"

	if err := envconfig.Process("", &cfg); err != nil {
		return fmt.Errorf("processing config: %w", err)
	}

	// -------------------------------------------------------------------------
	// App Info & Config Logging

	log.Info(ctx, "startup", "version", cfg.Version)
	log.Info(ctx, "startup", "config", sanitizeConfig(cfg))

	// -------------------------------------------------------------------------
	// App Starting

	log.Info(ctx, "starting service", "version", cfg.Version.Build)
	defer log.Info(ctx, "shutdown complete")

	log.BuildInfo(ctx)

	expvar.NewString("build").Set(cfg.Version.Build)

	// -------------------------------------------------------------------------
	// Database Support

	log.Info(ctx, "startup", "status", "initializing database support", "hostport", cfg.DB.Host)

	db, err := sqldb.Open(sqldb.Config{
		User:         cfg.DB.User,
		Password:     cfg.DB.Password,
		Host:         cfg.DB.Host,
		Name:         cfg.DB.Name,
		MaxIdleConns: cfg.DB.MaxIdleConns,
		MaxOpenConns: cfg.DB.MaxOpenConns,
		DisableTLS:   cfg.DB.DisableTLS,
	})
	if err != nil {
		return fmt.Errorf("connecting to db: %w", err)
	}

	defer db.Close()

	// -------------------------------------------------------------------------
	// Auth Support

	log.Info(ctx, "startup", "status", "initializing authentication support")

	ks := keystore.New()

	if _, err := ks.LoadByFileSystem(os.DirFS("foundation/zarf/keys")); err != nil {
		return fmt.Errorf("loading keys: %w", err)
	}

	// -------------------------------------------------------------------------
	// Start Tracing Support

	// Default: Tracing desabilitado (No-Op) para evitar nil pointers
	log.Info(ctx, "startup", "status", "initializing tracing support")

	traceProvider, teardown, err := otel.InitTracing(log, otel.Config{
		ServiceName: "",
		Host:        cfg.Tempo.Host,
		ExcludedRoutes: map[string]struct{}{
			"/v1/liveness":  {},
			"/v1/readiness": {},
		},
		Probability: cfg.Tempo.Probability,
	})
	if err != nil {
		return fmt.Errorf("starting tracing: %w", err)
	}

	defer teardown(context.Background())

	tracer := traceProvider.Tracer(cfg.Tempo.ServiceName)

	log.Info(ctx, "startup", "status", "initializing V1 API support")

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	cfgMux := mux.Config{
		Build:      cfg.Version.Build,
		Log:        log,
		DB:         db,
		Tracer:     tracer,
		AuthConfig: mux.AuthConfig{ks, "https://spiexata.com/auth/"},
	}

	webAPI := mux.WebAPI(cfgMux,
		buildRoutes(), // Corrigido de build.Routes()
		mux.WithCORS(cfg.Web.CORSAllowedOrigins),
	)

	api := http.Server{
		Addr:         cfg.Web.APIHost,
		Handler:      webAPI,
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		IdleTimeout:  cfg.Web.IdleTimeout,
		ErrorLog:     logger.NewStdLogger(log, logger.LevelError),
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Info(ctx, "startup", "status", "api router started", "host", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	// -------------------------------------------------------------------------
	// Shutdown

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Info(ctx, "shutdown", "status", "shutdown started", "signal", sig)
		defer log.Info(ctx, "shutdown", "status", "shutdown complete", "signal", sig)

		ctx, cancel := context.WithTimeout(ctx, cfg.Web.ShutdownTimeout)
		defer cancel()

		if err := api.Shutdown(ctx); err != nil {
			api.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}

func buildRoutes() mux.RouteAdder {

	// The idea here is that we can build different versions of the binary
	// with different sets of exposed web APIs. By default we build a single
	// instance with all the web APIs.
	//
	// Here is the scenario. It would be nice to build two binaries, one for the
	// transactional APIs (CRUD) and one for the reporting APIs. This would allow
	// the system to run two instances of the database. One instance tuned for the
	// transactional database calls and the other tuned for the reporting calls.
	// Tuning meaning indexing and memory requirements. The two databases can be
	// kept in sync with replication.
	//switch routes {
	//case "crud":
	// return crud.Routes()
	//case "reporting":
	// return reporting.Routes()
	// }

	return all.Routes()
}

func sanitizeConfig(cfg Config) string {
	cfg.DB.Password = "[MASKED]"

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Sprintf("%+v", cfg)
	}
	return string(data)
}
