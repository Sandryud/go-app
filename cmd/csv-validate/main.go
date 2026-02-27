// Command csv-validate checks exercise CSV files in a directory for errors.
// Directory path is set via -csv flag. Optional -catalog runs catalog validation after parsing CSV.
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"workout-app/internal/exercises/csvparser"
	"workout-app/internal/exercises/csvvalidator"
	"workout-app/internal/exercises/validator"
)

func main() {
	csvDir := flag.String("csv", "data/csv", "Directory containing CSV files to validate")
	catalog := flag.Bool("catalog", false, "After CSV validation, parse and run catalog validator")
	flag.Parse()

	ctx := context.Background()
	csvErrs, err := csvvalidator.Validate(ctx, *csvDir)
	if err != nil {
		log.Fatalf("Read error: %v", err)
	}

	if len(csvErrs) > 0 {
		log.Printf("CSV validation found %d error(s)", len(csvErrs))
		for _, e := range csvErrs {
			log.Printf("  • %s", e.String())
		}
		log.Printf("Validation failed.")
		os.Exit(1)
	}

	log.Printf("CSV validation passed.")

	if *catalog {
		log.Printf("Parsing catalog and running catalog validation...")
		catalog, parseErr := csvparser.Run(ctx, *csvDir)
		if parseErr != nil {
			log.Fatalf("Parse error: %v", parseErr)
		}
		catalogErrs := validator.Validate(catalog)
		if len(catalogErrs) > 0 {
			log.Printf("Catalog validation found %d error(s)", len(catalogErrs))
			for _, s := range catalogErrs {
				log.Printf("  • %s", s)
			}
			log.Printf("Catalog validation failed.")
			os.Exit(1)
		}
		log.Printf("Catalog validation passed.")
	}
}
