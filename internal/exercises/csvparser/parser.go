// Package csvparser reads exercise CSV files and builds a single JSON catalog.
package csvparser

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"workout-app/data/examples"
)

// Run reads all CSVs from dir and returns a Catalog. Errors include file and context.
// ctx is propagated for cancellation and future I/O timeouts.
func Run(ctx context.Context, dir string) (*examples.Catalog, error) {
	p := &parser{dir: dir}
	return p.run(ctx)
}

type parser struct {
	dir string

	// Reference tables: id -> slug (or unit string)
	equipment        map[int]string
	movementPatterns map[int]string
	purposes         map[int]string
	tags             map[int]string
	skills           map[int]string
	measurementUnits map[int]string

	// Link tables: exercise_id -> list of ref IDs (order preserved)
	equipmentByExercise        map[string][]int
	movementPatternsByExercise map[string][]int
	purposesByExercise         map[string][]int
	tagsByExercise             map[string][]int
	skillsByExercise           map[string][]int
	measurementUnitsByExercise  map[string][]int

	// One-to-many by exercise_id
	instructions      map[string][]string // ordered by step_number
	muscles           map[string][]examples.MuscleActivation
	media             map[string][]examples.MediaAsset
	warnings          map[string][]examples.Warning
	contraindications map[string][]examples.Contraindication
	mistakes          map[string][]string
	programmingNotes  map[string][]string
	breathingTips     map[string][]string
	references        map[string][]examples.ExerciseReference

	// Type-specific rows keyed by exercise_id
	strengthByExercise map[string]*strengthRow
	cardioByExercise   map[string]*cardioRow
	mobilityByExercise map[string]*mobilityRow
}

type strengthRow struct {
	force         string
	breathing     examples.Breathing
	programming   *examples.Programming
}

type cardioRow struct {
	breathing   examples.Breathing
	programming *examples.Programming
}

type mobilityRow struct {
	programming *examples.Programming
}

func (p *parser) run(ctx context.Context) (*examples.Catalog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := p.loadRefs(); err != nil {
		return nil, err
	}
	if err := p.loadLinks(); err != nil {
		return nil, err
	}
	if err := p.loadOneToMany(); err != nil {
		return nil, err
	}
	if err := p.loadTypeSpecific(); err != nil {
		return nil, err
	}

	exercises, _, err := p.loadExercises()
	if err != nil {
		return nil, err
	}

	// Merge nested data into each exercise
	for i := range exercises {
		e := &exercises[i]
		p.mergeEquipment(e)
		p.mergeMovementPatterns(e)
		p.mergePurposes(e)
		p.mergeTags(e)
		p.mergeSkills(e)
		p.mergeMeasurementUnits(e)
		e.Instructions = p.instructions[e.ID]
		e.Muscles = p.muscles[e.ID]
		e.Media = p.media[e.ID]
		e.Warnings = p.warnings[e.ID]
		e.Contraindications = p.contraindications[e.ID]
		e.Mistakes = p.mistakes[e.ID]
		e.ProgrammingNotes = p.programmingNotes[e.ID]
		p.mergeBreathing(e)
		e.References = p.references[e.ID]
		p.mergeTypeSpecific(e)
	}

	now := time.Now().UTC()
	return &examples.Catalog{
		Meta: examples.Meta{
			Version:     now.Format("20060102150405"), // YYYYMMDDHHmmss, e.g. 20260227120100
			GeneratedAt: now.Format(time.RFC3339),
			TotalCount:  len(exercises),
		},
		Exercises: exercises,
	}, nil
}

func readCSV(path, name string) ([]string, [][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("file %s: %w", name, err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("file %s: read: %w", name, err)
	}
	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("file %s: missing required column header", name)
	}
	return rows[0], rows[1:], nil
}

func colIndex(headers []string, name string) (int, error) {
	for i, h := range headers {
		if h == name {
			return i, nil
		}
	}
	return -1, fmt.Errorf("missing required column %q", name)
}

// optionalColIndex returns column index and true if name exists, or -1, false if not. Use for optional columns.
func optionalColIndex(headers []string, name string) (int, bool) {
	for i, h := range headers {
		if h == name {
			return i, true
		}
	}
	return -1, false
}

func at(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return row[i]
}

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

func parseJSONCell(context, raw string, v any) error {
	if raw == "" {
		return nil
	}
	err := json.Unmarshal([]byte(raw), v)
	if err != nil {
		return fmt.Errorf("%s: json: %w", context, err)
	}
	return nil
}
