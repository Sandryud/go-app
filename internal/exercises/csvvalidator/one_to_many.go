package csvvalidator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"workout-app/internal/exercises/enums"
)

// validateOneToMany validates one-to-many CSV files. Missing files are skipped.
// Returns an error if any existing file fails to read (I/O error).
func validateOneToMany(ctx context.Context, dir string, exerciseIDs map[string]bool, errs *[]Error) error {
	files := []struct {
		name     string
		validate func(string, []string, [][]string, map[string]bool, *[]Error)
	}{
		{"exercise_instructions.csv", validateExerciseInstructions},
		{"muscle_activations.csv", validateMuscleActivations},
		{"media_assets.csv", validateMediaAssets},
		{"warnings.csv", validateWarnings},
		{"contraindications.csv", validateContraindications},
		{"exercise_mistakes.csv", validateExerciseMistakes},
		{"programming_notes.csv", validateProgrammingNotes},
		{"breathing_tips.csv", validateBreathingTips},
		{"exercise_references.csv", validateExerciseReferences},
	}
	var ioErr error
	for _, f := range files {
		path := filepath.Join(dir, f.name)
		headers, rows, err := readCSV(ctx, path, f.name)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			*errs = append(*errs, Error{File: f.name, Row: 1, Message: err.Error()})
			ioErr = errors.Join(ioErr, fmt.Errorf("%s: %w", f.name, err))
			continue
		}
		f.validate(f.name, headers, rows, exerciseIDs, errs)
	}
	return ioErr
}

func validateExerciseInstructions(name string, headers []string, rows [][]string, exerciseIDs map[string]bool, errs *[]Error) {
	idxEx, e := colIndex(headers, "exercise_id")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxStep, e := colIndex(headers, "step_number")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxInst, e := colIndex(headers, "instruction")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	for i, row := range rows {
		rowNum := i + 2
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		if !exerciseIDs[eid] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "exercise_id", Message: "exercise_id " + eid + " not found in exercises.csv"})
		}
		if _, err := parseInt(at(row, idxStep)); err != nil {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "step_number", Message: err.Error()})
		}
		if at(row, idxInst) == "" {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "instruction", Message: "required"})
		}
	}
}

func validateMuscleActivations(name string, headers []string, rows [][]string, exerciseIDs map[string]bool, errs *[]Error) {
	idxEx, e := colIndex(headers, "exercise_id")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxType, e := colIndex(headers, "muscle_type")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxAct, e := colIndex(headers, "activation_level")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	for i, row := range rows {
		rowNum := i + 2
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		if !exerciseIDs[eid] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "exercise_id", Message: "exercise_id " + eid + " not found in exercises.csv"})
		}
		mt := at(row, idxType)
		if mt != "" && !enums.MuscleTypes[mt] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "muscle_type", Message: "must be primary|secondary|stabilizer"})
		}
		v, err := parseInt(at(row, idxAct))
		if err != nil {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "activation_level", Message: err.Error()})
		} else if v < 0 || v > 100 {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "activation_level", Message: "must be between 0 and 100"})
		}
	}
}

func validateMediaAssets(name string, headers []string, rows [][]string, exerciseIDs map[string]bool, errs *[]Error) {
	idxEx, e := colIndex(headers, "exercise_id")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxURL, e := colIndex(headers, "url")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxType, e := colIndex(headers, "type")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	for i, row := range rows {
		rowNum := i + 2
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		if !exerciseIDs[eid] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "exercise_id", Message: "exercise_id " + eid + " not found in exercises.csv"})
		}
		if at(row, idxURL) == "" {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "url", Message: "required"})
		}
		mt := at(row, idxType)
		if !enums.MediaTypes[mt] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "type", Message: "must be image|video"})
		}
	}
}

func validateWarnings(name string, headers []string, rows [][]string, exerciseIDs map[string]bool, errs *[]Error) {
	idxEx, e := colIndex(headers, "exercise_id")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxLevel, e := colIndex(headers, "level")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxMsg, e := colIndex(headers, "message")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxRec, _ := optionalColIndex(headers, "recommendations")
	for i, row := range rows {
		rowNum := i + 2
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		if !exerciseIDs[eid] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "exercise_id", Message: "exercise_id " + eid + " not found in exercises.csv"})
		}
		lvl := at(row, idxLevel)
		if !enums.WarningLevels[lvl] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "level", Message: "must be critical|important|notice"})
		}
		if at(row, idxMsg) == "" {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "message", Message: "required"})
		}
		if idxRec >= 0 {
			raw := at(row, idxRec)
			if raw != "" && !isValidJSON(raw) {
				*errs = append(*errs, Error{File: name, Row: rowNum, Column: "recommendations", Message: "invalid JSON"})
			}
		}
	}
}

func validateContraindications(name string, headers []string, rows [][]string, exerciseIDs map[string]bool, errs *[]Error) {
	idxEx, e := colIndex(headers, "exercise_id")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxSev, e := colIndex(headers, "severity")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxAlt, e := colIndex(headers, "alternatives")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	for i, row := range rows {
		rowNum := i + 2
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		if !exerciseIDs[eid] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "exercise_id", Message: "exercise_id " + eid + " not found in exercises.csv"})
		}
		sev := at(row, idxSev)
		if !enums.Severities[sev] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "severity", Message: "must be absolute|relative"})
		}
		raw := at(row, idxAlt)
		if raw != "" && !isValidJSON(raw) {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "alternatives", Message: "invalid JSON"})
		}
	}
}

func validateExerciseMistakes(name string, headers []string, rows [][]string, exerciseIDs map[string]bool, errs *[]Error) {
	idxEx, e := colIndex(headers, "exercise_id")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxMistake, e := colIndex(headers, "mistake")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	for i, row := range rows {
		rowNum := i + 2
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		if !exerciseIDs[eid] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "exercise_id", Message: "exercise_id " + eid + " not found in exercises.csv"})
		}
		if at(row, idxMistake) == "" {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "mistake", Message: "required"})
		}
	}
}

func validateProgrammingNotes(name string, headers []string, rows [][]string, exerciseIDs map[string]bool, errs *[]Error) {
	idxEx, e := colIndex(headers, "exercise_id")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxNote, e := colIndex(headers, "note")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	for i, row := range rows {
		rowNum := i + 2
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		if !exerciseIDs[eid] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "exercise_id", Message: "exercise_id " + eid + " not found in exercises.csv"})
		}
		if at(row, idxNote) == "" {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "note", Message: "required"})
		}
	}
}

func validateBreathingTips(name string, headers []string, rows [][]string, exerciseIDs map[string]bool, errs *[]Error) {
	idxEx, e := colIndex(headers, "exercise_id")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxTip, e := colIndex(headers, "tip")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	for i, row := range rows {
		rowNum := i + 2
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		if !exerciseIDs[eid] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "exercise_id", Message: "exercise_id " + eid + " not found in exercises.csv"})
		}
		if at(row, idxTip) == "" {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "tip", Message: "required"})
		}
	}
}

// refLink is a comparable key for a single reference (from, to, relationship).
type refLink struct {
	From string
	To   string
	Rel  string
}

// inverseRelationship returns the relationship type that must appear on the reverse link.
// progression <-> regression, alternative <-> alternative.
func inverseRelationship(rel string) string {
	switch rel {
	case "progression":
		return "regression"
	case "regression":
		return "progression"
	case "alternative":
		return "alternative"
	default:
		return ""
	}
}

func validateExerciseReferences(name string, headers []string, rows [][]string, exerciseIDs map[string]bool, errs *[]Error) {
	idxFrom, e := colIndex(headers, "from_exercise_id")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxTo, e := colIndex(headers, "to_exercise_id")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxRel, e := colIndex(headers, "relationship")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}
	idxEff, e := colIndex(headers, "effectiveness_rating")
	if e != nil {
		*errs = append(*errs, Error{File: name, Row: 1, Message: e.Error()})
		return
	}

	// First pass: validate fields and build set of existing links (from, to, rel).
	links := make(map[refLink]bool)
	for i, row := range rows {
		rowNum := i + 2
		fromID := at(row, idxFrom)
		if fromID == "" {
			continue
		}
		if !exerciseIDs[fromID] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "from_exercise_id", Message: "from_exercise_id " + fromID + " not found in exercises.csv"})
		}
		toID := at(row, idxTo)
		if toID != "" && !exerciseIDs[toID] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "to_exercise_id", Message: "to_exercise_id " + toID + " not found in exercises.csv"})
		}
		rel := at(row, idxRel)
		if !enums.Relationships[rel] {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "relationship", Message: "must be regression|progression|alternative"})
		}
		v, err := parseInt(at(row, idxEff))
		if err != nil {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "effectiveness_rating", Message: err.Error()})
		} else if v < 0 || v > 100 {
			*errs = append(*errs, Error{File: name, Row: rowNum, Column: "effectiveness_rating", Message: "must be between 0 and 100"})
		}
		if toID != "" && enums.Relationships[rel] && exerciseIDs[fromID] && exerciseIDs[toID] {
			links[refLink{From: fromID, To: toID, Rel: rel}] = true
		}
	}

	// Second pass: check every link has a reverse link (only for rows with valid from/to in exercises).
	for i, row := range rows {
		rowNum := i + 2
		fromID := at(row, idxFrom)
		toID := at(row, idxTo)
		rel := at(row, idxRel)
		if fromID == "" || toID == "" || !enums.Relationships[rel] || !exerciseIDs[fromID] || !exerciseIDs[toID] {
			continue
		}
		invRel := inverseRelationship(rel)
		if invRel == "" {
			continue
		}
		reverse := refLink{From: toID, To: fromID, Rel: invRel}
		if !links[reverse] {
			*errs = append(*errs, Error{
				File:    name,
				Row:     rowNum,
				Column:  "reverse_link",
				Message: "link from " + fromID + " to " + toID + " (" + rel + ") missing reverse: expected row from " + toID + " to " + fromID + " with relationship=" + invRel,
			})
		}
	}
}
