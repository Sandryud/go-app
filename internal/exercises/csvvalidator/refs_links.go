package csvvalidator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// refTableResult holds loaded reference IDs and whether the ref file existed.
type refTableResult struct {
	ids   map[int]bool
	loaded bool
}

func validateRefsAndLinks(ctx context.Context, dir string, exerciseIDs map[string]bool, errs *[]Error) error {
	equipment := loadRefTable(ctx, dir, "exercise_equipment.csv", "id", "equipment", errs)
	patterns := loadRefTable(ctx, dir, "movement_patterns.csv", "id", "pattern", errs)
	purposes := loadRefTable(ctx, dir, "exercise_purposes.csv", "id", "purpose", errs)
	tags := loadRefTable(ctx, dir, "exercise_tags.csv", "id", "tag", errs)
	skills := loadRefTable(ctx, dir, "exercise_skills.csv", "id", "skill", errs)
	units := loadRefTable(ctx, dir, "measurement_units.csv", "id", "unit", errs)

	ioErr := validateLinkTable(ctx, dir, "exercise_equipment_links.csv", "exercise_equipment.csv", "exercise_id", "equipment_id", exerciseIDs, equipment, errs)
	ioErr = errors.Join(ioErr, validateLinkTable(ctx, dir, "exercise_movement_pattern_links.csv", "movement_patterns.csv", "exercise_id", "pattern_id", exerciseIDs, patterns, errs))
	ioErr = errors.Join(ioErr, validateLinkTable(ctx, dir, "exercise_purpose_links.csv", "exercise_purposes.csv", "exercise_id", "purpose_id", exerciseIDs, purposes, errs))
	ioErr = errors.Join(ioErr, validateLinkTable(ctx, dir, "exercise_tag_links.csv", "exercise_tags.csv", "exercise_id", "tag_id", exerciseIDs, tags, errs))
	ioErr = errors.Join(ioErr, validateLinkTable(ctx, dir, "exercise_skill_links.csv", "exercise_skills.csv", "exercise_id", "skill_id", exerciseIDs, skills, errs))
	ioErr = errors.Join(ioErr, validateLinkTable(ctx, dir, "exercise_measurement_unit_links.csv", "measurement_units.csv", "exercise_id", "unit_id", exerciseIDs, units, errs))
	return ioErr
}

// loadRefTable loads a reference CSV and returns its IDs. When the file is missing, returns (nil, false).
// On I/O or parse error, appends to errs and returns (nil, false).
func loadRefTable(ctx context.Context, dir, filename, idCol, valueCol string, errs *[]Error) refTableResult {
	path := filepath.Join(dir, filename)
	headers, rows, err := readCSV(ctx, path, filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return refTableResult{loaded: false}
		}
		*errs = append(*errs, Error{File: filename, Row: 1, Message: err.Error()})
		return refTableResult{loaded: false}
	}
	idxID, e := colIndex(headers, idCol)
	if e != nil {
		*errs = append(*errs, Error{File: filename, Row: 1, Message: e.Error()})
		return refTableResult{loaded: true, ids: nil}
	}
	_, e = colIndex(headers, valueCol)
	if e != nil {
		*errs = append(*errs, Error{File: filename, Row: 1, Message: e.Error()})
		return refTableResult{loaded: true, ids: nil}
	}
	ids := make(map[int]bool)
	for i, row := range rows {
		rowNum := i + 2
		id, err := parseInt(at(row, idxID))
		if err != nil {
			*errs = append(*errs, Error{File: filename, Row: rowNum, Column: idCol, Message: err.Error()})
			continue
		}
		if id > 0 {
			ids[id] = true
		}
	}
	return refTableResult{ids: ids, loaded: true}
}

// validateLinkTable validates a link CSV. When the link file exists but the ref table was not loaded,
// appends an error that the reference table is required and skips FK validation.
// Returns an I/O error when the link file exists but could not be read.
func validateLinkTable(ctx context.Context, dir, filename, refFilename, exCol, fkCol string, exerciseIDs map[string]bool, ref refTableResult, errs *[]Error) error {
	path := filepath.Join(dir, filename)
	headers, rows, err := readCSV(ctx, path, filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		*errs = append(*errs, Error{File: filename, Row: 1, Message: err.Error()})
		return fmt.Errorf("%s: %w", filename, err)
	}
	effectiveRefIDs := ref.ids
	if !ref.loaded {
		*errs = append(*errs, Error{File: filename, Row: 1, Message: "reference table " + refFilename + " required for link table " + filename})
		effectiveRefIDs = make(map[int]bool) // empty so every FK fails validation
	}
	idxEx, e := colIndex(headers, exCol)
	if e != nil {
		*errs = append(*errs, Error{File: filename, Row: 1, Message: e.Error()})
		return nil
	}
	idxFK, e := colIndex(headers, fkCol)
	if e != nil {
		*errs = append(*errs, Error{File: filename, Row: 1, Message: e.Error()})
		return nil
	}
	for i, row := range rows {
		rowNum := i + 2
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		if !exerciseIDs[eid] {
			*errs = append(*errs, Error{File: filename, Row: rowNum, Column: exCol, Message: exCol + " " + eid + " not found in exercises.csv"})
		}
		if effectiveRefIDs != nil {
			fk, err := parseInt(at(row, idxFK))
			if err != nil {
				*errs = append(*errs, Error{File: filename, Row: rowNum, Column: fkCol, Message: err.Error()})
			} else if fk > 0 && !effectiveRefIDs[fk] {
				*errs = append(*errs, Error{File: filename, Row: rowNum, Column: fkCol, Message: fkCol + " " + at(row, idxFK) + " not found in reference table"})
			}
		}
	}
	return nil
}
