package csvparser

import (
	"fmt"
	"path/filepath"

	"workout-app/data/examples"
)

func (p *parser) loadRefs() error {
	// equipment: id -> equipment (slug)
	headers, rows, err := readCSV(filepath.Join(p.dir, "exercise_equipment.csv"), "exercise_equipment.csv")
	if err != nil {
		return err
	}
	idxID, err := colIndex(headers, "id")
	if err != nil {
		return fmt.Errorf("exercise_equipment.csv: %w", err)
	}
	idxSlug, err := colIndex(headers, "equipment")
	if err != nil {
		return fmt.Errorf("exercise_equipment.csv: %w", err)
	}
	p.equipment = make(map[int]string)
	for _, row := range rows {
		id, _ := parseInt(at(row, idxID))
		if id > 0 {
			p.equipment[id] = at(row, idxSlug)
		}
	}

	// movement_patterns: id -> pattern
	headers, rows, err = readCSV(filepath.Join(p.dir, "movement_patterns.csv"), "movement_patterns.csv")
	if err != nil {
		return err
	}
	idxID, err = colIndex(headers, "id")
	if err != nil {
		return fmt.Errorf("movement_patterns.csv: %w", err)
	}
	idxSlug, err = colIndex(headers, "pattern")
	if err != nil {
		return fmt.Errorf("movement_patterns.csv: %w", err)
	}
	p.movementPatterns = make(map[int]string)
	for _, row := range rows {
		id, _ := parseInt(at(row, idxID))
		if id > 0 {
			p.movementPatterns[id] = at(row, idxSlug)
		}
	}

	// exercise_purposes: id -> purpose
	headers, rows, err = readCSV(filepath.Join(p.dir, "exercise_purposes.csv"), "exercise_purposes.csv")
	if err != nil {
		return err
	}
	idxID, err = colIndex(headers, "id")
	if err != nil {
		return fmt.Errorf("exercise_purposes.csv: %w", err)
	}
	idxSlug, err = colIndex(headers, "purpose")
	if err != nil {
		return fmt.Errorf("exercise_purposes.csv: %w", err)
	}
	p.purposes = make(map[int]string)
	for _, row := range rows {
		id, _ := parseInt(at(row, idxID))
		if id > 0 {
			p.purposes[id] = at(row, idxSlug)
		}
	}

	// exercise_tags: id -> tag
	headers, rows, err = readCSV(filepath.Join(p.dir, "exercise_tags.csv"), "exercise_tags.csv")
	if err != nil {
		return err
	}
	idxID, err = colIndex(headers, "id")
	if err != nil {
		return fmt.Errorf("exercise_tags.csv: %w", err)
	}
	idxSlug, err = colIndex(headers, "tag")
	if err != nil {
		return fmt.Errorf("exercise_tags.csv: %w", err)
	}
	p.tags = make(map[int]string)
	for _, row := range rows {
		id, _ := parseInt(at(row, idxID))
		if id > 0 {
			p.tags[id] = at(row, idxSlug)
		}
	}

	// exercise_skills: id -> skill
	headers, rows, err = readCSV(filepath.Join(p.dir, "exercise_skills.csv"), "exercise_skills.csv")
	if err != nil {
		return err
	}
	idxID, err = colIndex(headers, "id")
	if err != nil {
		return fmt.Errorf("exercise_skills.csv: %w", err)
	}
	idxSlug, err = colIndex(headers, "skill")
	if err != nil {
		return fmt.Errorf("exercise_skills.csv: %w", err)
	}
	p.skills = make(map[int]string)
	for _, row := range rows {
		id, _ := parseInt(at(row, idxID))
		if id > 0 {
			p.skills[id] = at(row, idxSlug)
		}
	}

	// measurement_units: id -> unit
	headers, rows, err = readCSV(filepath.Join(p.dir, "measurement_units.csv"), "measurement_units.csv")
	if err != nil {
		return err
	}
	idxID, err = colIndex(headers, "id")
	if err != nil {
		return fmt.Errorf("measurement_units.csv: %w", err)
	}
	idxSlug, err = colIndex(headers, "unit")
	if err != nil {
		return fmt.Errorf("measurement_units.csv: %w", err)
	}
	p.measurementUnits = make(map[int]string)
	for _, row := range rows {
		id, _ := parseInt(at(row, idxID))
		if id > 0 {
			p.measurementUnits[id] = at(row, idxSlug)
		}
	}

	return nil
}

func (p *parser) loadLinks() error {
	// equipment_links: exercise_id -> []equipment_id
	headers, rows, err := readCSV(filepath.Join(p.dir, "exercise_equipment_links.csv"), "exercise_equipment_links.csv")
	if err != nil {
		return err
	}
	idxEx, err := colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("exercise_equipment_links.csv: %w", err)
	}
	idxEq, err := colIndex(headers, "equipment_id")
	if err != nil {
		return fmt.Errorf("exercise_equipment_links.csv: %w", err)
	}
	p.equipmentByExercise = make(map[string][]int)
	for _, row := range rows {
		eid := at(row, idxEx)
		eqID, _ := parseInt(at(row, idxEq))
		if eid != "" && eqID > 0 {
			p.equipmentByExercise[eid] = append(p.equipmentByExercise[eid], eqID)
		}
	}

	// movement_pattern_links
	headers, rows, err = readCSV(filepath.Join(p.dir, "exercise_movement_pattern_links.csv"), "exercise_movement_pattern_links.csv")
	if err != nil {
		return err
	}
	idxEx, err = colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("exercise_movement_pattern_links.csv: %w", err)
	}
	idxPat, err := colIndex(headers, "pattern_id")
	if err != nil {
		return fmt.Errorf("exercise_movement_pattern_links.csv: %w", err)
	}
	p.movementPatternsByExercise = make(map[string][]int)
	for _, row := range rows {
		eid := at(row, idxEx)
		pid, _ := parseInt(at(row, idxPat))
		if eid != "" && pid > 0 {
			p.movementPatternsByExercise[eid] = append(p.movementPatternsByExercise[eid], pid)
		}
	}

	// purpose_links
	headers, rows, err = readCSV(filepath.Join(p.dir, "exercise_purpose_links.csv"), "exercise_purpose_links.csv")
	if err != nil {
		return err
	}
	idxEx, err = colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("exercise_purpose_links.csv: %w", err)
	}
	idxPurp, err := colIndex(headers, "purpose_id")
	if err != nil {
		return fmt.Errorf("exercise_purpose_links.csv: %w", err)
	}
	p.purposesByExercise = make(map[string][]int)
	for _, row := range rows {
		eid := at(row, idxEx)
		pid, _ := parseInt(at(row, idxPurp))
		if eid != "" && pid > 0 {
			p.purposesByExercise[eid] = append(p.purposesByExercise[eid], pid)
		}
	}

	// tag_links
	headers, rows, err = readCSV(filepath.Join(p.dir, "exercise_tag_links.csv"), "exercise_tag_links.csv")
	if err != nil {
		return err
	}
	idxEx, err = colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("exercise_tag_links.csv: %w", err)
	}
	idxTag, err := colIndex(headers, "tag_id")
	if err != nil {
		return fmt.Errorf("exercise_tag_links.csv: %w", err)
	}
	p.tagsByExercise = make(map[string][]int)
	for _, row := range rows {
		eid := at(row, idxEx)
		tid, _ := parseInt(at(row, idxTag))
		if eid != "" && tid > 0 {
			p.tagsByExercise[eid] = append(p.tagsByExercise[eid], tid)
		}
	}

	// skill_links
	headers, rows, err = readCSV(filepath.Join(p.dir, "exercise_skill_links.csv"), "exercise_skill_links.csv")
	if err != nil {
		return err
	}
	idxEx, err = colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("exercise_skill_links.csv: %w", err)
	}
	idxSkill, err := colIndex(headers, "skill_id")
	if err != nil {
		return fmt.Errorf("exercise_skill_links.csv: %w", err)
	}
	p.skillsByExercise = make(map[string][]int)
	for _, row := range rows {
		eid := at(row, idxEx)
		sid, _ := parseInt(at(row, idxSkill))
		if eid != "" && sid > 0 {
			p.skillsByExercise[eid] = append(p.skillsByExercise[eid], sid)
		}
	}

	// measurement_unit_links
	headers, rows, err = readCSV(filepath.Join(p.dir, "exercise_measurement_unit_links.csv"), "exercise_measurement_unit_links.csv")
	if err != nil {
		return err
	}
	idxEx, err = colIndex(headers, "exercise_id")
	if err != nil {
		return fmt.Errorf("exercise_measurement_unit_links.csv: %w", err)
	}
	idxUnit, err := colIndex(headers, "unit_id")
	if err != nil {
		return fmt.Errorf("exercise_measurement_unit_links.csv: %w", err)
	}
	p.measurementUnitsByExercise = make(map[string][]int)
	for _, row := range rows {
		eid := at(row, idxEx)
		uid, _ := parseInt(at(row, idxUnit))
		if eid != "" && uid > 0 {
			p.measurementUnitsByExercise[eid] = append(p.measurementUnitsByExercise[eid], uid)
		}
	}

	return nil
}

func (p *parser) mergeEquipment(e *examples.Exercise) {
	ids := p.equipmentByExercise[e.ID]
	for _, id := range ids {
		if s, ok := p.equipment[id]; ok {
			e.Equipment = append(e.Equipment, s)
		}
	}
}

func (p *parser) mergeMovementPatterns(e *examples.Exercise) {
	ids := p.movementPatternsByExercise[e.ID]
	for _, id := range ids {
		if s, ok := p.movementPatterns[id]; ok {
			e.MovementPatterns = append(e.MovementPatterns, s)
		}
	}
}

func (p *parser) mergePurposes(e *examples.Exercise) {
	ids := p.purposesByExercise[e.ID]
	for _, id := range ids {
		if s, ok := p.purposes[id]; ok {
			e.Purposes = append(e.Purposes, s)
		}
	}
}

func (p *parser) mergeTags(e *examples.Exercise) {
	ids := p.tagsByExercise[e.ID]
	for _, id := range ids {
		if s, ok := p.tags[id]; ok {
			e.Tags = append(e.Tags, s)
		}
	}
}

func (p *parser) mergeSkills(e *examples.Exercise) {
	ids := p.skillsByExercise[e.ID]
	for _, id := range ids {
		if s, ok := p.skills[id]; ok {
			e.Skills = append(e.Skills, s)
		}
	}
}

func (p *parser) mergeMeasurementUnits(e *examples.Exercise) {
	ids := p.measurementUnitsByExercise[e.ID]
	for _, id := range ids {
		if s, ok := p.measurementUnits[id]; ok {
			e.MeasurementUnits = append(e.MeasurementUnits, s)
		}
	}
}
