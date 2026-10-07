package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/strydio/governed-marketing-data-mcp/internal/evaluator"
	"github.com/strydio/governed-marketing-data-mcp/internal/runs"
)

func main() {
	var inputPath string
	var groundTruthPath string
	flag.StringVar(&inputPath, "input", "", "path to client run record JSON")
	flag.StringVar(&groundTruthPath, "ground-truth", "./testdata/ground_truth.json", "path to ground truth JSON")
	flag.Parse()

	if inputPath == "" {
		fmt.Fprintln(os.Stderr, "--input is required")
		os.Exit(1)
	}

	contents, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read input: %v\n", err)
		os.Exit(1)
	}
	var record runs.Record
	if err := json.Unmarshal(contents, &record); err != nil {
		fmt.Fprintf(os.Stderr, "parse input: %v\n", err)
		os.Exit(1)
	}

	groundTruth, err := evaluator.LoadGroundTruth(groundTruthPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load ground truth: %v\n", err)
		os.Exit(1)
	}

	result := evaluator.Evaluate(record, groundTruth)
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode result: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
}
