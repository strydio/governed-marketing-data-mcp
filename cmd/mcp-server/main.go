package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/strydio/governed-marketing-data-mcp/internal/repository"
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
		fmt.Println("governed marketing data MCP server")
	}
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
