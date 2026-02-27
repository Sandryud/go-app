package csvparser

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"

	"workout-app/data/examples"
)

type instructionRow struct {
	exerciseID string
	stepNum    int
	text       string
}

func (p *parser) loadOneToMany() error {
	p.instructions = make(map[string][]string)
	p.muscles = make(map[string][]examples.MuscleActivation)
	p.media = make(map[string][]examples.MediaAsset)
	p.warnings = make(map[string][]examples.Warning)
	p.contraindications = make(map[string][]examples.Contraindication)
	p.mistakes = make(map[string][]string)
	p.programmingNotes = make(map[string][]string)
	p.breathingTips = make(map[string][]string)
	p.references = make(map[string][]examples.ExerciseReference)

	// instructions: sort by step_number
	headers, rows, err := readCSV(filepath.Join(p.dir, "exercise_instructions.csv"), "exercise_instructions.csv")
	if err != nil {
		return err
	}
	idxEx, err := colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("exercise_instructions.csv: %w", err)
	}
	idxStep, err := colIndex(headers, "step_number")
	if err != nil {
		return fmt.Errorf("exercise_instructions.csv: %w", err)
	}
	idxInst, err := colIndex(headers, "instruction")
	if err != nil {
		return fmt.Errorf("exercise_instructions.csv: %w", err)
	}
	var instRows []instructionRow
	for _, row := range rows {
		eid := at(row, idxEx)
		step, _ := parseInt(at(row, idxStep))
		instRows = append(instRows, instructionRow{eid, step, at(row, idxInst)})
	}
	sort.Slice(instRows, func(i, j int) bool {
		if instRows[i].exerciseID != instRows[j].exerciseID {
			return instRows[i].exerciseID < instRows[j].exerciseID
		}
		return instRows[i].stepNum < instRows[j].stepNum
	})
	for _, r := range instRows {
		if r.exerciseID != "" && r.text != "" {
			p.instructions[r.exerciseID] = append(p.instructions[r.exerciseID], r.text)
		}
	}

	// muscle_activations
	headers, rows, err = readCSV(filepath.Join(p.dir, "muscle_activations.csv"), "muscle_activations.csv")
	if err != nil {
		return err
	}
	idxEx, err = colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("muscle_activations.csv: %w", err)
	}
	idxType, err := colIndex(headers, "muscle_type")
	if err != nil {
		return fmt.Errorf("muscle_activations.csv: %w", err)
	}
	idxName, err := colIndex(headers, "muscle_name")
	if err != nil {
		return fmt.Errorf("muscle_activations.csv: %w", err)
	}
	idxAct, err := colIndex(headers, "activation_level")
	if err != nil {
		return fmt.Errorf("muscle_activations.csv: %w", err)
	}
	for _, row := range rows {
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		act, _ := parseInt(at(row, idxAct))
		p.muscles[eid] = append(p.muscles[eid], examples.MuscleActivation{
			Type:       at(row, idxType),
			Name:       at(row, idxName),
			Activation: act,
		})
	}

	// media_assets: alt_text -> alt, duration -> duration_s
	headers, rows, err = readCSV(filepath.Join(p.dir, "media_assets.csv"), "media_assets.csv")
	if err != nil {
		return err
	}
	idxEx, err = colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("media_assets.csv: %w", err)
	}
	idxURL, err := colIndex(headers, "url")
	if err != nil {
		return fmt.Errorf("media_assets.csv: %w", err)
	}
	idxType, err = colIndex(headers, "type")
	if err != nil {
		return fmt.Errorf("media_assets.csv: %w", err)
	}
	idxAlt, err := colIndex(headers, "alt_text")
	if err != nil {
		return fmt.Errorf("media_assets.csv: %w", err)
	}
	idxDur, err := colIndex(headers, "duration")
	if err != nil {
		return fmt.Errorf("media_assets.csv: %w", err)
	}
	idxThumb, err := colIndex(headers, "thumbnail_url")
	if err != nil {
		return fmt.Errorf("media_assets.csv: %w", err)
	}
	idxPrimary, err := colIndex(headers, "is_primary")
	if err != nil {
		return fmt.Errorf("media_assets.csv: %w", err)
	}
	idxSort, err := colIndex(headers, "sort_order")
	if err != nil {
		return fmt.Errorf("media_assets.csv: %w", err)
	}
	for _, row := range rows {
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		primary, _ := parseBool(at(row, idxPrimary))
		sortOrder, _ := parseInt(at(row, idxSort))
		dur, _ := parseInt(at(row, idxDur))
		p.media[eid] = append(p.media[eid], examples.MediaAsset{
			URL:          at(row, idxURL),
			Type:         at(row, idxType),
			Alt:          at(row, idxAlt),
			DurationSec:  dur,
			ThumbnailURL: at(row, idxThumb),
			IsPrimary:    primary,
			SortOrder:    sortOrder,
		})
	}

	// warnings: related_body_part -> body_part, recommendations JSON
	headers, rows, err = readCSV(filepath.Join(p.dir, "warnings.csv"), "warnings.csv")
	if err != nil {
		return err
	}
	idxEx, err = colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("warnings.csv: %w", err)
	}
	idxLevel, err := colIndex(headers, "level")
	if err != nil {
		return fmt.Errorf("warnings.csv: %w", err)
	}
	idxMsg, err := colIndex(headers, "message")
	if err != nil {
		return fmt.Errorf("warnings.csv: %w", err)
	}
	idxBody, err := colIndex(headers, "related_body_part")
	if err != nil {
		return fmt.Errorf("warnings.csv: %w", err)
	}
	idxRec, err := colIndex(headers, "recommendations")
	if err != nil {
		return fmt.Errorf("warnings.csv: %w", err)
	}
	for i, row := range rows {
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		w := examples.Warning{
			Level:    at(row, idxLevel),
			Message:  at(row, idxMsg),
			BodyPart: at(row, idxBody),
		}
		raw := at(row, idxRec)
		if raw != "" {
			var rec struct {
				Recommendations []string `json:"recommendations"`
			}
			if err := parseJSONCell("warnings.csv row "+strconv.Itoa(i+2)+", recommendations", raw, &rec); err != nil {
				return err
			}
			w.Recommendations = rec.Recommendations
		}
		p.warnings[eid] = append(p.warnings[eid], w)
	}

	// contraindications: alternatives JSON
	headers, rows, err = readCSV(filepath.Join(p.dir, "contraindications.csv"), "contraindications.csv")
	if err != nil {
		return err
	}
	idxEx, err = colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("contraindications.csv: %w", err)
	}
	idxCond, err := colIndex(headers, "condition")
	if err != nil {
		return fmt.Errorf("contraindications.csv: %w", err)
	}
	idxSev, err := colIndex(headers, "severity")
	if err != nil {
		return fmt.Errorf("contraindications.csv: %w", err)
	}
	idxReason, err := colIndex(headers, "reason")
	if err != nil {
		return fmt.Errorf("contraindications.csv: %w", err)
	}
	idxAlt, err = colIndex(headers, "alternatives")
	if err != nil {
		return fmt.Errorf("contraindications.csv: %w", err)
	}
	for i, row := range rows {
		eid := at(row, idxEx)
		if eid == "" {
			continue
		}
		c := examples.Contraindication{
			Condition: at(row, idxCond),
			Severity:  at(row, idxSev),
			Reason:    at(row, idxReason),
		}
		raw := at(row, idxAlt)
		if raw != "" {
			var alt struct {
				Alternatives []string `json:"alternatives"`
			}
			if err := parseJSONCell("contraindications.csv row "+strconv.Itoa(i+2)+", alternatives", raw, &alt); err != nil {
				return err
			}
			c.Alternatives = alt.Alternatives
		}
		p.contraindications[eid] = append(p.contraindications[eid], c)
	}

	// exercise_mistakes
	headers, rows, err = readCSV(filepath.Join(p.dir, "exercise_mistakes.csv"), "exercise_mistakes.csv")
	if err != nil {
		return err
	}
	idxEx, err = colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("exercise_mistakes.csv: %w", err)
	}
	idxMistake, err := colIndex(headers, "mistake")
	if err != nil {
		return fmt.Errorf("exercise_mistakes.csv: %w", err)
	}
	for _, row := range rows {
		eid := at(row, idxEx)
		m := at(row, idxMistake)
		if eid != "" && m != "" {
			p.mistakes[eid] = append(p.mistakes[eid], m)
		}
	}

	// programming_notes
	headers, rows, err = readCSV(filepath.Join(p.dir, "programming_notes.csv"), "programming_notes.csv")
	if err != nil {
		return err
	}
	idxEx, err = colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("programming_notes.csv: %w", err)
	}
	idxNote, err := colIndex(headers, "note")
	if err != nil {
		return fmt.Errorf("programming_notes.csv: %w", err)
	}
	for _, row := range rows {
		eid := at(row, idxEx)
		n := at(row, idxNote)
		if eid != "" && n != "" {
			p.programmingNotes[eid] = append(p.programmingNotes[eid], n)
		}
	}

	// breathing_tips
	headers, rows, err = readCSV(filepath.Join(p.dir, "breathing_tips.csv"), "breathing_tips.csv")
	if err != nil {
		return err
	}
	idxEx, err = colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("breathing_tips.csv: %w", err)
	}
	idxTip, err := colIndex(headers, "tip")
	if err != nil {
		return fmt.Errorf("breathing_tips.csv: %w", err)
	}
	for _, row := range rows {
		eid := at(row, idxEx)
		t := at(row, idxTip)
		if eid != "" && t != "" {
			p.breathingTips[eid] = append(p.breathingTips[eid], t)
		}
	}

	// exercise_references: from_exercise_id -> refs (exercise_id = to_exercise_id)
	headers, rows, err = readCSV(filepath.Join(p.dir, "exercise_references.csv"), "exercise_references.csv")
	if err != nil {
		return err
	}
	idxFrom, err := colIndex(headers, "from_exercise_id")
	if err != nil {
		return fmt.Errorf("exercise_references.csv: %w", err)
	}
	idxTo, err := colIndex(headers, "to_exercise_id")
	if err != nil {
		return fmt.Errorf("exercise_references.csv: %w", err)
	}
	idxName, err = colIndex(headers, "name")
	if err != nil {
		return fmt.Errorf("exercise_references.csv: %w", err)
	}
	idxRel, err := colIndex(headers, "relationship")
	if err != nil {
		return fmt.Errorf("exercise_references.csv: %w", err)
	}
	idxEff, err := colIndex(headers, "effectiveness_rating")
	if err != nil {
		return fmt.Errorf("exercise_references.csv: %w", err)
	}
	for _, row := range rows {
		fromID := at(row, idxFrom)
		if fromID == "" {
			continue
		}
		eff, _ := parseInt(at(row, idxEff))
		p.references[fromID] = append(p.references[fromID], examples.ExerciseReference{
			ExerciseID:          at(row, idxTo),
			Relationship:        at(row, idxRel),
			Name:                at(row, idxName),
			EffectivenessRating: eff,
		})
	}

	return nil
}
