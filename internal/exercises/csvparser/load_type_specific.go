package csvparser

import (
	"fmt"
	"path/filepath"
	"strconv"

	"workout-app/data/examples"
)

// programming JSON in strength/cardio/mobility CSVs
type programmingJSON struct {
	Sets      *minMax `json:"sets"`
	Reps      *minMax `json:"reps"`
	Rest      *minMax `json:"rest"`
	Tempo     string  `json:"tempo"`
	Intensity *minMax `json:"intensity"`
	Hold      *minMax `json:"hold"`
}

type minMax struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

func (p *parser) loadTypeSpecific() error {
	p.strengthByExercise = make(map[string]*strengthRow)
	p.cardioByExercise = make(map[string]*cardioRow)
	p.mobilityByExercise = make(map[string]*mobilityRow)

	// strength_exercises
	headers, rows, err := readCSV(filepath.Join(p.dir, "strength_exercises.csv"), "strength_exercises.csv")
	if err != nil {
		return err
	}
	idxEx, err := colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("strength_exercises.csv: %w", err)
	}
	idxForce, err := colIndex(headers, "force")
	if err != nil {
		return fmt.Errorf("strength_exercises.csv: %w", err)
	}
	idxEcc, err := colIndex(headers, "breathing_eccentric")
	if err != nil {
		return fmt.Errorf("strength_exercises.csv: %w", err)
	}
	idxConc, err := colIndex(headers, "breathing_concentric")
	if err != nil {
		return fmt.Errorf("strength_exercises.csv: %w", err)
	}
	idxDesc, err := colIndex(headers, "breathing_description")
	if err != nil {
		return fmt.Errorf("strength_exercises.csv: %w", err)
	}
	idxProg, err := colIndex(headers, "programming")
	if err != nil {
		return fmt.Errorf("strength_exercises.csv: %w", err)
	}
	for i, row := range rows {
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		sr := &strengthRow{
			force: at(row, idxForce),
			breathing: examples.Breathing{
				Eccentric:   at(row, idxEcc),
				Concentric:  at(row, idxConc),
				Description: at(row, idxDesc),
			},
		}
		raw := at(row, idxProg)
		if raw != "" {
			var prog programmingJSON
			if err := parseJSONCell("strength_exercises.csv row "+strconv.Itoa(i+2)+", programming", raw, &prog); err != nil {
				return err
			}
			sr.programming = programmingToStruct(prog)
		}
		sr.breathing.Tips = p.breathingTips[eid]
		p.strengthByExercise[eid] = sr
	}

	// cardio_exercises (may have empty rows)
	headers, rows, err = readCSV(filepath.Join(p.dir, "cardio_exercises.csv"), "cardio_exercises.csv")
	if err != nil {
		return err
	}
	idxEx, err = colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("cardio_exercises.csv: %w", err)
	}
	idxEcc, err = colIndex(headers, "breathing_eccentric")
	if err != nil {
		return fmt.Errorf("cardio_exercises.csv: %w", err)
	}
	idxConc, err = colIndex(headers, "breathing_concentric")
	if err != nil {
		return fmt.Errorf("cardio_exercises.csv: %w", err)
	}
	idxDesc, err = colIndex(headers, "breathing_description")
	if err != nil {
		return fmt.Errorf("cardio_exercises.csv: %w", err)
	}
	idxProg, err = colIndex(headers, "programming")
	if err != nil {
		return fmt.Errorf("cardio_exercises.csv: %w", err)
	}
	for i, row := range rows {
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		cr := &cardioRow{
			breathing: examples.Breathing{
				Eccentric:   at(row, idxEcc),
				Concentric:  at(row, idxConc),
				Description: at(row, idxDesc),
			},
		}
		cr.breathing.Tips = p.breathingTips[eid]
		raw := at(row, idxProg)
		if raw != "" {
			var prog programmingJSON
			if err := parseJSONCell("cardio_exercises.csv row "+strconv.Itoa(i+2)+", programming", raw, &prog); err != nil {
				return err
			}
			cr.programming = programmingToStruct(prog)
		}
		p.cardioByExercise[eid] = cr
	}

	// mobility_exercises
	headers, rows, err = readCSV(filepath.Join(p.dir, "mobility_exercises.csv"), "mobility_exercises.csv")
	if err != nil {
		return err
	}
	idxEx, err = colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("mobility_exercises.csv: %w", err)
	}
	idxProg, err = colIndex(headers, "programming")
	if err != nil {
		return fmt.Errorf("mobility_exercises.csv: %w", err)
	}
	for i, row := range rows {
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		mr := &mobilityRow{}
		raw := at(row, idxProg)
		if raw != "" {
			var prog programmingJSON
			if err := parseJSONCell("mobility_exercises.csv row "+strconv.Itoa(i+2)+", programming", raw, &prog); err != nil {
				return err
			}
			mr.programming = programmingToStruct(prog)
		}
		p.mobilityByExercise[eid] = mr
	}

	return nil
}

func programmingToStruct(prog programmingJSON) *examples.Programming {
	p := &examples.Programming{Tempo: prog.Tempo}
	if prog.Sets != nil {
		p.SetsMin, p.SetsMax = prog.Sets.Min, prog.Sets.Max
	}
	if prog.Reps != nil {
		p.RepsMin, p.RepsMax = prog.Reps.Min, prog.Reps.Max
	}
	if prog.Rest != nil {
		p.RestSecMin, p.RestSecMax = prog.Rest.Min, prog.Rest.Max
	}
	if prog.Intensity != nil {
		p.IntensityPctMin, p.IntensityPctMax = prog.Intensity.Min, prog.Intensity.Max
	}
	if prog.Hold != nil {
		p.HoldSecMin, p.HoldSecMax = prog.Hold.Min, prog.Hold.Max
	}
	return p
}

func (p *parser) mergeBreathing(e *examples.Exercise) {
	switch e.Type {
	case "strength":
		if sr := p.strengthByExercise[e.ID]; sr != nil {
			e.Breathing = &sr.breathing
		}
	case "cardio":
		if cr := p.cardioByExercise[e.ID]; cr != nil {
			e.Breathing = &cr.breathing
		}
	}
}

func (p *parser) mergeTypeSpecific(e *examples.Exercise) {
	switch e.Type {
	case "strength":
		if sr := p.strengthByExercise[e.ID]; sr != nil {
			e.Strength = &examples.StrengthParams{Force: sr.force, Programming: sr.programming}
		}
	case "cardio":
		if cr := p.cardioByExercise[e.ID]; cr != nil {
			e.Cardio = &examples.CardioParams{Programming: cr.programming}
		}
	case "mobility":
		if mr := p.mobilityByExercise[e.ID]; mr != nil {
			e.Mobility = &examples.MobilityParams{Programming: mr.programming}
		}
	}
}
