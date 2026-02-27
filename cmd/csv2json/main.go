// Command csv2json reads exercise CSV files from a directory and writes
// a single JSON catalog. Intended for CI/CD.
// Validation runs by default; use -validate=false to skip (e.g. when debugging).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"

	"workout-app/data/examples"
	"workout-app/internal/exercises/csvparser"
	"workout-app/internal/exercises/validator"
)

func main() {
	csvDir := flag.String("csv", "data/csv", "Directory containing CSV files")
	outPath := flag.String("out", "dist/exercises.json", "Output JSON file path")
	validate := flag.Bool("validate", true, "Run validator before writing (use -validate=false to skip)")
	flag.Parse()

	catalog, err := csvparser.Run(context.Background(), *csvDir)
	if err != nil {
		log.Fatalf("parse: %v", err)
	}

	if *validate {
		errs := validator.Validate(catalog)
		if len(errs) > 0 {
			for _, e := range errs {
				log.Printf("validation: %s", e)
			}
			os.Exit(1)
		}
	}

	if err := writeJSON(*outPath, catalog); err != nil {
		log.Fatalf("write: %v", err)
	}
	log.Printf("wrote %s (%d exercises)", *outPath, catalog.Meta.TotalCount)
}

func writeJSON(path string, c *examples.Catalog) (err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err = enc.Encode(c); err != nil {
		return err
	}
	return nil
}
