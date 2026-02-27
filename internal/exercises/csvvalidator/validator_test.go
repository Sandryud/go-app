package csvvalidator

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_ValidExercisesOnly(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"ex-1", "strength", "Bench", "beginner", "chest", "weight-reps", "gym"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	assert.Empty(t, errs)
}

func TestValidate_InvalidDifficulty(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"ex-1", "strength", "Bench", "super", "chest", "weight-reps", "gym"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "beginner|intermediate|advanced|expert")
	assert.Equal(t, "difficulty", errs[0].Column)
}

func TestValidate_InvalidType(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"ex-1", "flex", "Bench", "beginner", "chest", "weight-reps", "gym"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "strength|cardio|mobility|plyometric")
	assert.Equal(t, "type", errs[0].Column)
}

func TestValidate_InvalidLocation(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"ex-1", "strength", "Bench", "beginner", "chest", "weight-reps", "outdoor"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "gym|home|both")
}

func TestValidate_DuplicateID(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"dup", "strength", "First", "beginner", "chest", "weight-reps", "gym"},
		{"dup", "strength", "Second", "beginner", "back", "weight-reps", "gym"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "duplicate")
	assert.Equal(t, "id", errs[0].Column)
}

func TestValidate_EmptyID(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"", "strength", "No ID", "beginner", "chest", "weight-reps", "gym"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "required")
	assert.Equal(t, "id", errs[0].Column)
}

func TestValidate_InvalidPopularity(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location", "popularity_score"},
		{"ex-1", "strength", "Bench", "beginner", "chest", "weight-reps", "gym", "150"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "0 and 100")
	assert.Equal(t, "popularity_score", errs[0].Column)
}

func TestValidate_InvalidMetadataJSON(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location", "metadata"},
		{"ex-1", "strength", "Bench", "beginner", "chest", "weight-reps", "gym", "{invalid json"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "invalid JSON")
	assert.Equal(t, "metadata", errs[0].Column)
}

func TestValidate_OneToMany_UnknownExerciseID(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"ex-1", "strength", "Bench", "beginner", "chest", "weight-reps", "gym"},
	})
	writeCSV(t, dir, "exercise_instructions.csv", [][]string{
		{"exercise_id", "step_number", "instruction"},
		{"ex-1", "1", "Do step 1"},
		{"nonexistent", "1", "Orphan step"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "not found in exercises.csv")
	assert.Equal(t, "exercise_instructions.csv", errs[0].File)
	assert.Equal(t, "exercise_id", errs[0].Column)
}

func TestValidate_References_ToExerciseIDNotFound(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"ex-1", "strength", "Bench", "beginner", "chest", "weight-reps", "gym"},
	})
	writeCSV(t, dir, "exercise_references.csv", [][]string{
		{"from_exercise_id", "to_exercise_id", "name", "relationship", "effectiveness_rating"},
		{"ex-1", "nonexistent-ref", "Other", "progression", "80"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "not found in exercises.csv")
	assert.Equal(t, "exercise_references.csv", errs[0].File)
	assert.Equal(t, "to_exercise_id", errs[0].Column)
}

func TestValidate_References_InvalidRelationship(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"ex-1", "strength", "A", "beginner", "chest", "weight-reps", "gym"},
		{"ex-2", "strength", "B", "beginner", "chest", "weight-reps", "gym"},
	})
	writeCSV(t, dir, "exercise_references.csv", [][]string{
		{"from_exercise_id", "to_exercise_id", "name", "relationship", "effectiveness_rating"},
		{"ex-1", "ex-2", "B", "sibling", "80"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "regression|progression|alternative")
	assert.Equal(t, "relationship", errs[0].Column)
}

func TestValidate_References_Bidirectional_Valid(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"push-ups", "strength", "Отжимания", "beginner", "chest", "bodyweight-reps", "both"},
		{"bench-press", "strength", "Жим лежа", "intermediate", "chest", "weight-reps", "gym"},
	})
	writeCSV(t, dir, "exercise_references.csv", [][]string{
		{"from_exercise_id", "to_exercise_id", "name", "relationship", "effectiveness_rating"},
		{"push-ups", "bench-press", "Жим лежа", "progression", "85"},
		{"bench-press", "push-ups", "Отжимания", "regression", "82"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	assert.Empty(t, errs)
}

func TestValidate_References_Bidirectional_MissingReverse(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"ex-a", "strength", "A", "beginner", "chest", "weight-reps", "gym"},
		{"ex-b", "strength", "B", "intermediate", "chest", "weight-reps", "gym"},
	})
	writeCSV(t, dir, "exercise_references.csv", [][]string{
		{"from_exercise_id", "to_exercise_id", "name", "relationship", "effectiveness_rating"},
		{"ex-a", "ex-b", "B", "progression", "80"},
		// missing: ex-b -> ex-a with relationship=regression
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Equal(t, "exercise_references.csv", errs[0].File)
	assert.Equal(t, "reverse_link", errs[0].Column)
	assert.Contains(t, errs[0].Message, "missing reverse")
	assert.Contains(t, errs[0].Message, "ex-a")
	assert.Contains(t, errs[0].Message, "ex-b")
	assert.Contains(t, errs[0].Message, "progression")
	assert.Contains(t, errs[0].Message, "relationship=regression")
}

func TestValidate_References_Bidirectional_Alternative(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"ex-1", "strength", "X", "beginner", "chest", "weight-reps", "gym"},
		{"ex-2", "strength", "Y", "beginner", "chest", "weight-reps", "gym"},
	})
	writeCSV(t, dir, "exercise_references.csv", [][]string{
		{"from_exercise_id", "to_exercise_id", "name", "relationship", "effectiveness_rating"},
		{"ex-1", "ex-2", "Y", "alternative", "75"},
		{"ex-2", "ex-1", "X", "alternative", "75"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	assert.Empty(t, errs)
}

func TestValidate_RefsLinks_InvalidFK(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"ex-1", "strength", "Bench", "beginner", "chest", "weight-reps", "gym"},
	})
	writeCSV(t, dir, "exercise_equipment.csv", [][]string{
		{"id", "equipment", "name"},
		{"1", "barbell", "Barbell"},
	})
	writeCSV(t, dir, "exercise_equipment_links.csv", [][]string{
		{"exercise_id", "equipment_id"},
		{"ex-1", "1"},
		{"ex-1", "999"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Equal(t, "exercise_equipment_links.csv", errs[0].File)
	assert.Equal(t, "equipment_id", errs[0].Column)
	assert.Contains(t, errs[0].Message, "not found in reference table")
}

func TestValidate_RefsLinks_LinkFileWithoutRefFile(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"ex-1", "strength", "Bench", "beginner", "chest", "weight-reps", "gym"},
	})
	// exercise_equipment_links.csv exists but exercise_equipment.csv does not
	writeCSV(t, dir, "exercise_equipment_links.csv", [][]string{
		{"exercise_id", "equipment_id"},
		{"ex-1", "1"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(errs), 1)
	var found bool
	for _, e := range errs {
		if e.File == "exercise_equipment_links.csv" && e.Row == 1 && e.Message != "" {
			assert.Contains(t, e.Message, "reference table")
			assert.Contains(t, e.Message, "required")
			found = true
			break
		}
	}
	assert.True(t, found, "expected error about missing reference table")
}

func TestValidate_TypeSpecific_WrongExerciseType(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"ex-1", "cardio", "Run", "beginner", "legs", "time", "both"},
	})
	writeCSV(t, dir, "strength_exercises.csv", [][]string{
		{"exercise_id", "force"},
		{"ex-1", "push"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Equal(t, "strength_exercises.csv", errs[0].File)
	assert.Equal(t, "exercise_id", errs[0].Column)
	assert.Contains(t, errs[0].Message, "expected strength")
	assert.Contains(t, errs[0].Message, "cardio")
}

func TestValidate_TypeSpecific_InvalidForce(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"ex-1", "strength", "Bench", "beginner", "chest", "weight-reps", "gym"},
	})
	writeCSV(t, dir, "strength_exercises.csv", [][]string{
		{"exercise_id", "force"},
		{"ex-1", "lateral"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Equal(t, "strength_exercises.csv", errs[0].File)
	assert.Equal(t, "force", errs[0].Column)
	assert.Contains(t, errs[0].Message, "push|pull")
}

func TestValidate_TypeSpecific_InvalidProgrammingJSON(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "exercises.csv", [][]string{
		{"id", "type", "name", "difficulty", "primary_muscle_group", "tracking_type", "location"},
		{"ex-1", "strength", "Bench", "beginner", "chest", "weight-reps", "gym"},
	})
	writeCSV(t, dir, "strength_exercises.csv", [][]string{
		{"exercise_id", "force", "programming"},
		{"ex-1", "push", "{invalid"},
	})

	errs, err := Validate(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Equal(t, "strength_exercises.csv", errs[0].File)
	assert.Equal(t, "programming", errs[0].Column)
	assert.Contains(t, errs[0].Message, "invalid JSON")
}

func TestValidate_MissingExercisesCSV(t *testing.T) {
	dir := t.TempDir()
	// no exercises.csv

	_, err := Validate(context.Background(), dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exercises.csv")
}

func TestValidate_ErrorString(t *testing.T) {
	e := Error{File: "test.csv", Row: 5, Column: "id", Message: "required"}
	s := e.String()
	assert.Contains(t, s, "test.csv")
	assert.Contains(t, s, "5")
	assert.Contains(t, s, "id")
	assert.Contains(t, s, "required")
}

func writeCSV(t *testing.T, dir, name string, rows [][]string) {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	w := csv.NewWriter(f)
	for _, row := range rows {
		require.NoError(t, w.Write(row))
	}
	w.Flush()
	require.NoError(t, w.Error())
}
