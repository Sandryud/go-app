// Package validator checks exercise catalog for enums, required fields, and ranges.
package validator

import (
	"fmt"

	"workout-app/data/examples"
)

// ValidationError holds exercise_id, field, and message.
type ValidationError struct {
	ExerciseID string
	Field      string
	Message    string
}

func (e ValidationError) String() string {
	return fmt.Sprintf("exercise_id=%q field=%q message=%q", e.ExerciseID, e.Field, e.Message)
}

var (
	exerciseTypes     = map[string]bool{"strength": true, "cardio": true, "mobility": true, "plyometric": true}
	difficulties      = map[string]bool{"beginner": true, "intermediate": true, "advanced": true}
	locations         = map[string]bool{"gym": true, "home": true, "both": true}
	warningLevels      = map[string]bool{"critical": true, "important": true, "notice": true}
	severities         = map[string]bool{"absolute": true, "relative": true}
	relationships      = map[string]bool{"regression": true, "progression": true, "alternative": true}
	muscleTypes        = map[string]bool{"primary": true, "secondary": true, "stabilizer": true}
	mediaTypes         = map[string]bool{"image": true, "video": true}
	forces             = map[string]bool{"push": true, "pull": true}
)

// Validate returns a list of validation errors (empty if valid).
func Validate(c *examples.Catalog) []string {
	if c == nil {
		return []string{"catalog is nil"}
	}
	var out []string
	exerciseIDs := make(map[string]bool)
	for i := range c.Exercises {
		e := &c.Exercises[i]
		if e.ID != "" && exerciseIDs[e.ID] {
			out = append(out, ValidationError{e.ID, "id", "duplicate exercise id"}.String())
		}
		exerciseIDs[e.ID] = true
		out = append(out, validateExercise(e, exerciseIDs)...)
	}
	// Second pass: check references point to existing exercises
	for i := range c.Exercises {
		e := &c.Exercises[i]
		for _, ref := range e.References {
			if ref.ExerciseID != "" && !exerciseIDs[ref.ExerciseID] {
				out = append(out, ValidationError{e.ID, "references.exercise_id", "reference targets non-existent exercise " + ref.ExerciseID}.String())
			}
		}
	}
	return out
}

func validateExercise(e *examples.Exercise, allIDs map[string]bool) (errs []string) {
	exerciseID := e.ID
	if exerciseID == "" {
		errs = append(errs, ValidationError{exerciseID, "id", "required"}.String())
	}
	if !exerciseTypes[e.Type] {
		errs = append(errs, ValidationError{exerciseID, "type", "must be strength|cardio|mobility|plyometric"}.String())
	}
	if e.Name == "" {
		errs = append(errs, ValidationError{exerciseID, "name", "required"}.String())
	}
	if !difficulties[e.Difficulty] {
		errs = append(errs, ValidationError{exerciseID, "difficulty", "must be beginner|intermediate|advanced"}.String())
	}
	if e.PrimaryMuscleGroup == "" {
		errs = append(errs, ValidationError{exerciseID, "primary_muscle_group", "required"}.String())
	}
	if e.TrackingType == "" {
		errs = append(errs, ValidationError{exerciseID, "tracking_type", "required"}.String())
	}
	if !locations[e.Location] {
		errs = append(errs, ValidationError{exerciseID, "location", "must be gym|home|both"}.String())
	}
	if e.MinExperienceMonths < 0 {
		errs = append(errs, ValidationError{exerciseID, "min_experience_months", "must be >= 0"}.String())
	}
	if e.Popularity < 0 || e.Popularity > 100 {
		errs = append(errs, ValidationError{exerciseID, "popularity", "must be between 0 and 100"}.String())
	}
	for j, m := range e.Muscles {
		if !muscleTypes[m.Type] {
			errs = append(errs, ValidationError{exerciseID, fmt.Sprintf("muscles[%d].type", j), "must be primary|secondary|stabilizer"}.String())
		}
		if m.Activation < 0 || m.Activation > 100 {
			errs = append(errs, ValidationError{exerciseID, fmt.Sprintf("muscles[%d].activation", j), "must be between 0 and 100"}.String())
		}
	}
	for j, w := range e.Warnings {
		if !warningLevels[w.Level] {
			errs = append(errs, ValidationError{exerciseID, fmt.Sprintf("warnings[%d].level", j), "must be critical|important|notice"}.String())
		}
	}
	for j, c := range e.Contraindications {
		if !severities[c.Severity] {
			errs = append(errs, ValidationError{exerciseID, fmt.Sprintf("contraindications[%d].severity", j), "must be absolute|relative"}.String())
		}
	}
	for j, r := range e.References {
		if !relationships[r.Relationship] {
			errs = append(errs, ValidationError{exerciseID, fmt.Sprintf("references[%d].relationship", j), "must be regression|progression|alternative"}.String())
		}
		if r.EffectivenessRating < 0 || r.EffectivenessRating > 100 {
			errs = append(errs, ValidationError{exerciseID, fmt.Sprintf("references[%d].effectiveness_rating", j), "must be between 0 and 100"}.String())
		}
	}
	for j, m := range e.Media {
		if !mediaTypes[m.Type] {
			errs = append(errs, ValidationError{exerciseID, fmt.Sprintf("media[%d].type", j), "must be image|video"}.String())
		}
	}
	if e.Strength != nil {
		if e.Strength.Force != "" && !forces[e.Strength.Force] {
			errs = append(errs, ValidationError{exerciseID, "strength.force", "must be push|pull"}.String())
		}
		progErrs := validateProgramming(exerciseID, "strength.programming", e.Strength.Programming)
		errs = append(errs, progErrs...)
	}
	if e.Cardio != nil {
		progErrs := validateProgramming(exerciseID, "cardio.programming", e.Cardio.Programming)
		errs = append(errs, progErrs...)
	}
	if e.Mobility != nil {
		progErrs := validateProgramming(exerciseID, "mobility.programming", e.Mobility.Programming)
		errs = append(errs, progErrs...)
	}
	return errs
}

func validateProgramming(exerciseID, fieldPrefix string, p *examples.Programming) (errs []string) {
	if p == nil {
		return nil
	}
	if p.SetsMin < 0 || p.SetsMax < 0 || p.SetsMin > p.SetsMax {
		errs = append(errs, ValidationError{exerciseID, fieldPrefix + ".sets", "sets_min/sets_max must be non-negative and min <= max"}.String())
	}
	if p.RepsMin < 0 || p.RepsMax < 0 || p.RepsMin > p.RepsMax {
		errs = append(errs, ValidationError{exerciseID, fieldPrefix + ".reps", "reps_min/reps_max must be non-negative and min <= max"}.String())
	}
	if p.RestSecMin < 0 || p.RestSecMax < 0 || p.RestSecMin > p.RestSecMax {
		errs = append(errs, ValidationError{exerciseID, fieldPrefix + ".rest", "rest_sec_min/rest_sec_max must be non-negative and min <= max"}.String())
	}
	if p.IntensityPctMin < 0 || p.IntensityPctMin > 100 || p.IntensityPctMax < 0 || p.IntensityPctMax > 100 || p.IntensityPctMin > p.IntensityPctMax {
		if p.IntensityPctMin != 0 || p.IntensityPctMax != 0 {
			errs = append(errs, ValidationError{exerciseID, fieldPrefix + ".intensity", "intensity_pct_min/max must be 0-100 and min <= max"}.String())
		}
	}
	return errs
}
