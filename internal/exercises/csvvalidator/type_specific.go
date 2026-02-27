package csvvalidator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"workout-app/internal/exercises/enums"
)

// validateTypeSpecific validates type-specific CSV files (strength, cardio, mobility).
// Missing files are skipped. Returns an error if any existing file fails to read (I/O error).
func validateTypeSpecific(ctx context.Context, dir string, exerciseIDs map[string]bool, exerciseTypes map[string]string, errs *[]Error) error {
	ioErr := validateStrength(ctx, dir, exerciseIDs, exerciseTypes, errs)
	ioErr = errors.Join(ioErr, validateCardio(ctx, dir, exerciseIDs, exerciseTypes, errs))
	ioErr = errors.Join(ioErr, validateMobility(ctx, dir, exerciseIDs, exerciseTypes, errs))
	return ioErr
}

func validateStrength(ctx context.Context, dir string, exerciseIDs map[string]bool, exerciseTypes map[string]string, errs *[]Error) error {
	const name = "strength_exercises.csv"
	path := filepath.Join(dir, name)
	headers, rows, err := readCSV(ctx, path, name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		*errs = append(*errs, Error{File: name, Row: 1, Message: err.Error()})
		return fmt.Errorf("%s: %w", name, err)
	}
	idxEx, e := colIndex(headers, "exercise_id")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return nil
	}
	idxForce, e := colIndex(headers, "force")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return nil
	}
	idxProg, _ := optionalColIndex(headers, "programming")
	for i, row := range rows {
		rowNum := i + 2
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		if !exerciseIDs[eid] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "exercise_id", Message: "exercise_id " + eid + " not found in exercises.csv"})
		} else if exerciseTypes[eid] != "strength" {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "exercise_id", Message: "exercise " + eid + " has type " + exerciseTypes[eid] + " in exercises.csv, expected strength"})
		}
		force := at(row, idxForce)
		if force != "" && !enums.Forces[force] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "force", Message: "must be push|pull"})
		}
		if idxProg >= 0 {
			raw := at(row, idxProg)
			if raw != "" && !isValidJSON(raw) {
				*errs = append(*errs, Error{File: name, Row: rowNum, Column: "programming", Message: "invalid JSON"})
			}
		}
	}
	return nil
}

func validateCardio(ctx context.Context, dir string, exerciseIDs map[string]bool, exerciseTypes map[string]string, errs *[]Error) error {
	const name = "cardio_exercises.csv"
	path := filepath.Join(dir, name)
	headers, rows, err := readCSV(ctx, path, name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		*errs = append(*errs, Error{File: name, Row: 1, Message: err.Error()})
		return fmt.Errorf("%s: %w", name, err)
	}
	idxEx, e := colIndex(headers, "exercise_id")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return nil
	}
	idxProg, _ := optionalColIndex(headers, "programming")
	for i, row := range rows {
		rowNum := i + 2
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		if !exerciseIDs[eid] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "exercise_id", Message: "exercise_id " + eid + " not found in exercises.csv"})
		} else if exerciseTypes[eid] != "cardio" {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "exercise_id", Message: "exercise " + eid + " has type " + exerciseTypes[eid] + " in exercises.csv, expected cardio"})
		}
		if idxProg >= 0 {
			raw := at(row, idxProg)
			if raw != "" && !isValidJSON(raw) {
				*errs = append(*errs, Error{File: name, Row: rowNum, Column: "programming", Message: "invalid JSON"})
			}
		}
	}
	return nil
}

func validateMobility(ctx context.Context, dir string, exerciseIDs map[string]bool, exerciseTypes map[string]string, errs *[]Error) error {
	const name = "mobility_exercises.csv"
	path := filepath.Join(dir, name)
	headers, rows, err := readCSV(ctx, path, name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		*errs = append(*errs, Error{File: name, Row: 1, Message: err.Error()})
		return fmt.Errorf("%s: %w", name, err)
	}
	idxEx, e := colIndex(headers, "exercise_id")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return nil
	}
	idxProg, _ := optionalColIndex(headers, "programming")
	for i, row := range rows {
		rowNum := i + 2
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		if !exerciseIDs[eid] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "exercise_id", Message: "exercise_id " + eid + " not found in exercises.csv"})
		} else if exerciseTypes[eid] != "mobility" {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "exercise_id", Message: "exercise " + eid + " has type " + exerciseTypes[eid] + " in exercises.csv, expected mobility"})
		}
		if idxProg >= 0 {
			raw := at(row, idxProg)
			if raw != "" && !isValidJSON(raw) {
				*errs = append(*errs, Error{File: name, Row: rowNum, Column: "programming", Message: "invalid JSON"})
			}
		}
	}
	return nil
}
