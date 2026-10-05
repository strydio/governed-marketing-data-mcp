package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/strydio/governed-marketing-data-mcp/internal/client"
)

func main() {
	var cfg client.Config
	var outputPath string
	flag.StringVar(&cfg.Endpoint, "endpoint", envOrDefault("MCP_ENDPOINT", "http://localhost:8080/mcp"), "MCP Streamable HTTP endpoint")
	flag.StringVar(&cfg.Token, "token", os.Getenv("DEMO_BEARER_TOKEN"), "bearer token for the MCP server")
	flag.StringVar(&cfg.Campaign, "campaign", "autumn-sale", "campaign slug")
	flag.StringVar(&cfg.Market, "market", "UK", "market code")
	flag.StringVar(&cfg.StartDate, "start-date", "2026-09-14", "start date in YYYY-MM-DD format")
	flag.StringVar(&cfg.EndDate, "end-date", "2026-09-20", "end date in YYYY-MM-DD format")
	flag.StringVar(&outputPath, "output", "", "optional path to write the run record JSON")
	flag.Parse()

	record, err := client.RunDeterministic(context.Background(), cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "client failed: %v\n", err)
		os.Exit(1)
	}
	encoded, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode run record: %v\n", err)
		os.Exit(1)
	}
	if outputPath != "" {
		if err := os.WriteFile(outputPath, append(encoded, '\n'), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write output: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote run record to %s\n", outputPath)
		return
	}
	fmt.Println(string(encoded))
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
