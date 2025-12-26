package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/domain/tenantbus"
	tenantdb "github.com/jcpaschoal/spi-exata/business/domain/tenantbus/stores"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus/stores/usercache"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus/stores/userdb"
	"github.com/jcpaschoal/spi-exata/business/sdk/sqldb"
	"github.com/jcpaschoal/spi-exata/business/types/name"
	"github.com/jcpaschoal/spi-exata/business/types/password"
	"github.com/jcpaschoal/spi-exata/business/types/phone"
	"github.com/jcpaschoal/spi-exata/business/types/role"
	"github.com/jcpaschoal/spi-exata/foundation/logger"
	"github.com/kelseyhightower/envconfig"
)

// Config replicates necessary DB config structure
type Config struct {
	DB struct {
		User         string `envconfig:"DB_USER" default:"postgres"`
		Password     string `envconfig:"DB_PASSWORD" default:"postgres"`
		Host         string `envconfig:"DB_HOST" default:"localhost"`
		Name         string `envconfig:"DB_NAME" default:"spi"`
		MaxIdleConns int    `envconfig:"DB_MAX_IDLE_CONNS" default:"0"`
		MaxOpenConns int    `envconfig:"DB_MAX_OPEN_CONNS" default:"0"`
		DisableTLS   bool   `envconfig:"DB_DISABLE_TLS" default:"true"`
	}
}

func main() {
	log := logger.New(os.Stdout, logger.LevelInfo, "ADMIN-TOOL", nil)
	ctx := context.Background()

	if err := run(ctx, log); err != nil {
		log.Error(ctx, "startup", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *logger.Logger) error {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return fmt.Errorf("processing config: %w", err)
	}

	// Init DB
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

	// Init Domains
	userBus := userbus.NewCore(usercache.NewStore(log, userdb.NewStore(log, db), time.Minute))
	tenantBus := tenantbus.NewCore(log, tenantdb.NewStore(log, db))

	// CLI Parsing
	if len(os.Args) < 2 {
		fmt.Println("Usage: admin <command> [args]")
		fmt.Println("Commands: create-user, link-user")
		return nil
	}

	switch os.Args[1] {
	case "create-user":
		return runCreateUser(ctx, userBus, os.Args[2:])
	case "link-user":
		return runLinkUser(ctx, tenantBus, os.Args[2:])
	default:
		return fmt.Errorf("unknown command: %s", os.Args[1])
	}
}

func runCreateUser(ctx context.Context, ub *userbus.Core, args []string) error {
	cmd := flag.NewFlagSet("create-user", flag.ExitOnError)
	emailStr := cmd.String("email", "", "User email (Required)")
	passStr := cmd.String("password", "", "User password (Required)")
	nameStr := cmd.String("name", "", "User full name (Required)")
	roleStr := cmd.String("role", "USER", "User role (ADMIN, ANALYST, USER)")
	cmd.Parse(args)

	if *emailStr == "" || *passStr == "" || *nameStr == "" {
		cmd.PrintDefaults()
		return fmt.Errorf("missing required fields")
	}

	// Parsing Types using Domain Types
	n, err := name.Parse(*nameStr)
	if err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	r, err := role.Parse(*roleStr)
	if err != nil {
		return fmt.Errorf("invalid role: %w", err)
	}

	p, err := password.Parse(*passStr)
	if err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}

	newUser := userbus.NewUser{
		Name:     n,
		Email:    mailAddress(*emailStr),
		Password: p,
		Role:     r,
		Phone:    phone.Null{}, // Optional
	}

	usr, err := ub.Create(ctx, newUser)
	if err != nil {
		return fmt.Errorf("create user failed: %w", err)
	}

	fmt.Printf("\nSUCCESS: User created!\nID: %s\nEmail: %s\nRole: %s\n", usr.ID, usr.Email.Address, usr.Role)
	return nil
}

func runLinkUser(ctx context.Context, tb *tenantbus.Core, args []string) error {
	cmd := flag.NewFlagSet("link-user", flag.ExitOnError)
	userIDStr := cmd.String("user-id", "", "User UUID (Required)")
	dashIDStr := cmd.String("dashboard-id", "", "Dashboard UUID (Required)")
	cmd.Parse(args)

	if *userIDStr == "" || *dashIDStr == "" {
		cmd.PrintDefaults()
		return fmt.Errorf("missing required IDs")
	}

	userID, err := uuid.Parse(*userIDStr)
	if err != nil {
		return fmt.Errorf("invalid user uuid: %w", err)
	}

	dashID, err := uuid.Parse(*dashIDStr)
	if err != nil {
		return fmt.Errorf("invalid dashboard uuid: %w", err)
	}

	// Chama a nova função no tenantbus que orquestra a associação
	if err := tb.GrantUserAccessToDashboard(ctx, userID, dashID); err != nil {
		return fmt.Errorf("failed to link user: %w", err)
	}

	fmt.Printf("\nSUCCESS: User %s linked to Dashboard %s (and its Tenant)\n", userID, dashID)
	return nil
}

// Helper auxiliar para struct mail.Address, já que o construtor do pacote net/mail retorna ponteiro ou requer parsing complexo
func mailAddress(address string) struct{ Name, Address string } {
	return struct{ Name, Address string }{Address: address}
}

//go run api/tooling/admin/main.go create-user -email "admin@apexata.com" -password "Admin123!" -name "Admin User" -role "ADMIN"

//# Criar um Analista
//go run api/tooling/admin/main.go create-user -email "analista@apexata.com" -password "Analista123!" -name "Analyst User" -role "ANALYST"

//# Criar um Usuário Comum
//go run api/tooling/admin/main.go create-user -email "usuario@govsp.com" -password "User123!" -name "Normal User" -role "USER"

//go run api/tooling/admin/main.go create-user -email "usuario@govrj.com" -password "User123!" -name "Normal User" -role "USER"
