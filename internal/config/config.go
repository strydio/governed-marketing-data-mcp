package config

import (
	"fmt"
	"os"
)

type Config struct {
	MCPListenAddr           string
	MCPEndpoint             string
	DemoBearerToken         string
	SQLitePath              string
	PolicyPath              string
	MigrationsPath          string
	TelemetryCaptureContent bool
}

func Load() (Config, error) {
	cfg := Config{
		MCPListenAddr:   envOrDefault("MCP_LISTEN_ADDR", ":8080"),
		MCPEndpoint:     envOrDefault("MCP_ENDPOINT", "http://localhost:8080/mcp"),
		DemoBearerToken: os.Getenv("DEMO_BEARER_TOKEN"),
		SQLitePath:      envOrDefault("SQLITE_PATH", "./var/campaign.db"),
		PolicyPath:      envOrDefault("POLICY_PATH", "./config/policy.demo.yaml"),
		MigrationsPath:  envOrDefault("MIGRATIONS_PATH", "./migrations"),
	}
	cfg.TelemetryCaptureContent = os.Getenv("TELEMETRY_CAPTURE_CONTENT") == "true"

	if cfg.DemoBearerToken == "" {
		return Config{}, fmt.Errorf("DEMO_BEARER_TOKEN is required")
	}
	if cfg.PolicyPath == "" {
		return Config{}, fmt.Errorf("POLICY_PATH is required")
	}
	return cfg, nil
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
