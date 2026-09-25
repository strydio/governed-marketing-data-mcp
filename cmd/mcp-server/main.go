package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/strydio/governed-marketing-data-mcp/internal/authn"
	"github.com/strydio/governed-marketing-data-mcp/internal/authz"
	"github.com/strydio/governed-marketing-data-mcp/internal/config"
	"github.com/strydio/governed-marketing-data-mcp/internal/mcpserver"
	"github.com/strydio/governed-marketing-data-mcp/internal/repository"
	"github.com/strydio/governed-marketing-data-mcp/internal/telemetry"
)

func main() {
	var dbReset bool
	var dbSeed bool
	var sqlitePath string
	var migrationsPath string

	flag.BoolVar(&dbReset, "db-reset", false, "reset the local SQLite database")
	flag.BoolVar(&dbSeed, "db-seed", false, "apply schema and seed the local SQLite database")
	flag.StringVar(&sqlitePath, "sqlite-path", envOrDefault("SQLITE_PATH", "./var/campaign.db"), "path to the local SQLite database")
	flag.StringVar(&migrationsPath, "migrations-path", envOrDefault("MIGRATIONS_PATH", "./migrations"), "path to SQL migration files")
	flag.Parse()

	if dbReset {
		if err := repository.ResetDatabase(sqlitePath); err != nil {
			fmt.Fprintf(os.Stderr, "reset database: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("database reset complete")
	}

	if dbSeed {
		store, err := repository.OpenSQLite(sqlitePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open database: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		if err := store.ApplyMigrations(context.Background(), migrationsPath); err != nil {
			fmt.Fprintf(os.Stderr, "apply migrations: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("database schema and seed data applied")
	}

	if !dbReset && !dbSeed {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "load config: %v\n", err)
			os.Exit(1)
		}
		shutdownTelemetry, err := telemetry.InitOTLP(context.Background(), os.Getenv("OTEL_SERVICE_NAME"), os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "init telemetry: %v\n", err)
			os.Exit(1)
		}
		defer shutdownTelemetry(context.Background())

		store, err := repository.OpenSQLite(cfg.SQLitePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open database: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()
		authenticator, err := authn.NewAuthenticator(cfg.DemoBearerToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create authenticator: %v\n", err)
			os.Exit(1)
		}
		policy, err := authz.LoadPolicy(cfg.PolicyPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load policy: %v\n", err)
			os.Exit(1)
		}
		server := mcpserver.New(store, authenticator, policy)
		mux := http.NewServeMux()
		protectedHandler := mcpauth.RequireBearerToken(authenticator.VerifyToken, &mcpauth.RequireBearerTokenOptions{AllowMissingExpiration: true})(server.Handler())
		mux.Handle("/mcp", protectedHandler)
		fmt.Printf("governed marketing data MCP server listening on %s\n", cfg.MCPListenAddr)
		if err := http.ListenAndServe(cfg.MCPListenAddr, mux); err != nil {
			fmt.Fprintf(os.Stderr, "serve MCP: %v\n", err)
			os.Exit(1)
		}
	}
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
