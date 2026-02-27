package csvvalidator

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
)

// Validate reads all CSV files in dir and returns a list of validation errors.
// I/O errors (e.g. missing exercises.csv or read failure on optional files) are returned as the second value;
// logical validation errors are in the slice. Context is used for cancellation during file reads.
func Validate(ctx context.Context, dir string) ([]Error, error) {
	var errs []Error

	// Step 1: exercises.csv (required) — collect exercise IDs and types
	exerciseIDs, exerciseTypes, err := validateExercises(ctx, dir, &errs)
	if err != nil {
		return errs, err
	}

	// Step 2: one-to-many and references
	if err2 := validateOneToMany(ctx, dir, exerciseIDs, &errs); err2 != nil {
		err = errors.Join(err, err2)
	}

	// Step 3: reference tables and link tables
	if err2 := validateRefsAndLinks(ctx, dir, exerciseIDs, &errs); err2 != nil {
		err = errors.Join(err, err2)
	}

	// Step 4: type-specific (strength, cardio, mobility)
	if err2 := validateTypeSpecific(ctx, dir, exerciseIDs, exerciseTypes, &errs); err2 != nil {
		err = errors.Join(err, err2)
	}

	return errs, err
}

// readCSV reads a CSV file and returns headers and data rows. name is used in error messages.
// Supports context cancellation; only UTF-8 encoding is supported (see plan).
func readCSV(ctx context.Context, path, name string) (headers []string, rows [][]string, err error) {
	if ctx.Err() != nil {
		return nil, nil, fmt.Errorf("file %s: %w", name, ctx.Err())
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("file %s: %w", name, err)
	}
	defer f.Close()
	if ctx.Err() != nil {
		return nil, nil, fmt.Errorf("file %s: %w", name, ctx.Err())
	}
	r := csv.NewReader(f)
	all, err := r.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("file %s: read: %w", name, err)
	}
	if len(all) == 0 {
		return nil, nil, fmt.Errorf("file %s: missing header", name)
	}
	return all[0], all[1:], nil
}

// colIndex returns the index of the named column in headers, or an error if missing.
func colIndex(headers []string, name string) (int, error) {
	for i, h := range headers {
		if h == name {
			return i, nil
		}
	}
	return -1, fmt.Errorf("missing required column %q", name)
}

// optionalColIndex returns the index of the named column if present.
func optionalColIndex(headers []string, name string) (int, bool) {
	for i, h := range headers {
		if h == name {
			return i, true
		}
	}
	return -1, false
}

// at returns the i-th element of row, or empty string if out of range.
func at(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return row[i]
}

// parseInt parses s as an int; empty string is treated as 0.
func parseInt(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid int %q: %w", s, err)
	}
	return n, nil
}

// parseBool parses common boolean representations in CSV.
func parseBool(s string) (bool, error) {
	switch s {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no", "":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool %q", s)
	}
}

// isValidJSON reports whether s is valid JSON (or empty).
func isValidJSON(s string) bool {
	if s == "" {
		return true
	}
	var v any
	return json.Unmarshal([]byte(s), &v) == nil
}
