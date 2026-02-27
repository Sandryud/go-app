package csvvalidator

import (
	"context"
	"fmt"
	"path/filepath"

	"workout-app/internal/exercises/enums"
)

// validateExercises validates exercises.csv and returns exercise IDs and types (id -> type).
// Appends errors to errs. Returns (nil, nil, err) on I/O or missing required columns.
func validateExercises(ctx context.Context, dir string, errs *[]Error) (ids map[string]bool, types map[string]string, err error) {
	path := filepath.Join(dir, "exercises.csv")
	headers, rows, err := readCSV(ctx, path, "exercises.csv")
	if err != nil {
		return nil, nil, err
	}

	idxID, err := colIndex(headers, "id")
	if err != nil {
		return nil, nil, fmt.Errorf("exercises.csv: %w", err)
	}
	idxType, err := colIndex(headers, "type")
	if err != nil {
		return nil, nil, fmt.Errorf("exercises.csv: %w", err)
	}
	idxName, err := colIndex(headers, "name")
	if err != nil {
		return nil, nil, fmt.Errorf("exercises.csv: %w", err)
	}
	idxDiff, err := colIndex(headers, "difficulty")
	if err != nil {
		return nil, nil, fmt.Errorf("exercises.csv: %w", err)
	}
	idxMuscle, err := colIndex(headers, "primary_muscle_group")
	if err != nil {
		return nil, nil, fmt.Errorf("exercises.csv: %w", err)
	}
	idxTrack, err := colIndex(headers, "tracking_type")
	if err != nil {
		return nil, nil, fmt.Errorf("exercises.csv: %w", err)
	}
	idxLoc, err := colIndex(headers, "location")
	if err != nil {
		return nil, nil, fmt.Errorf("exercises.csv: %w", err)
	}
	idxMinExp, _ := optionalColIndex(headers, "minimum_experience")
	idxPop, _ := optionalColIndex(headers, "popularity_score")
	idxVerified, _ := optionalColIndex(headers, "is_verified")
	idxMeta, _ := optionalColIndex(headers, "metadata")

	ids = make(map[string]bool)
	types = make(map[string]string)
	seenID := make(map[string]bool)

	for i, row := range rows {
		rowNum := i + 2 // 1-based, header is row 1
		id := at(row, idxID)
		if id == "" {
			*errs = append(*errs, Error{File: "exercises.csv", Row: rowNum, Column: "id", Message: "required"})
			continue
		}
		if seenID[id] {
			*errs = append(*errs, Error{File: "exercises.csv", Row: rowNum, Column: "id", Message: "duplicate exercise id"})
		}
		seenID[id] = true
		ids[id] = true

		t := at(row, idxType)
		types[id] = t
		if !enums.ExerciseTypes[t] {
			*errs = append(*errs, Error{File: "exercises.csv", Row: rowNum, Column: "type", Message: "must be strength|cardio|mobility|plyometric"})
		}
		if at(row, idxName) == "" {
			*errs = append(*errs, Error{File: "exercises.csv", Row: rowNum, Column: "name", Message: "required"})
		}
		diff := at(row, idxDiff)
		if !enums.Difficulties[diff] {
			*errs = append(*errs, Error{File: "exercises.csv", Row: rowNum, Column: "difficulty", Message: "must be beginner|intermediate|advanced|expert"})
		}
		if at(row, idxMuscle) == "" {
			*errs = append(*errs, Error{File: "exercises.csv", Row: rowNum, Column: "primary_muscle_group", Message: "required"})
		}
		if at(row, idxTrack) == "" {
			*errs = append(*errs, Error{File: "exercises.csv", Row: rowNum, Column: "tracking_type", Message: "required"})
		}
		loc := at(row, idxLoc)
		if !enums.Locations[loc] {
			*errs = append(*errs, Error{File: "exercises.csv", Row: rowNum, Column: "location", Message: "must be gym|home|both"})
		}
		if idxMinExp >= 0 {
			v, e := parseInt(at(row, idxMinExp))
			if e != nil {
				*errs = append(*errs, Error{File: "exercises.csv", Row: rowNum, Column: "minimum_experience", Message: e.Error()})
			} else if v < 0 {
				*errs = append(*errs, Error{File: "exercises.csv", Row: rowNum, Column: "minimum_experience", Message: "must be >= 0"})
			}
		}
		if idxPop >= 0 {
			v, e := parseInt(at(row, idxPop))
			if e != nil {
				*errs = append(*errs, Error{File: "exercises.csv", Row: rowNum, Column: "popularity_score", Message: e.Error()})
			} else if v < 0 || v > 100 {
				*errs = append(*errs, Error{File: "exercises.csv", Row: rowNum, Column: "popularity_score", Message: "must be between 0 and 100"})
			}
		}
		if idxVerified >= 0 {
			_, e := parseBool(at(row, idxVerified))
			if e != nil {
				*errs = append(*errs, Error{File: "exercises.csv", Row: rowNum, Column: "is_verified", Message: e.Error()})
			}
		}
		if idxMeta >= 0 {
			raw := at(row, idxMeta)
			if raw != "" && !isValidJSON(raw) {
				*errs = append(*errs, Error{File: "exercises.csv", Row: rowNum, Column: "metadata", Message: "invalid JSON"})
			}
		}
	}

	return ids, types, nil
}
