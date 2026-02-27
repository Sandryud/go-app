package csvparser

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"workout-app/internal/exercises/validator"
)

// Integration test: run parser on data/csv and check structure and first exercise.
func TestRun_Integration(t *testing.T) {
	// From internal/exercises/csvparser, data/csv is ../../../data/csv
	csvDir := filepath.Join("..", "..", "..", "data", "csv")
	catalog, err := Run(context.Background(), csvDir)
	require.NoError(t, err)
	require.NotNil(t, catalog)
	assert.NotEmpty(t, catalog.Meta.Version)
	assert.NotEmpty(t, catalog.Meta.GeneratedAt)
	assert.Equal(t, len(catalog.Exercises), catalog.Meta.TotalCount)
	assert.GreaterOrEqual(t, len(catalog.Exercises), 1)

	first := catalog.Exercises[0]
	assert.Equal(t, "dumbbell-bench-press-flat", first.ID)
	assert.Equal(t, "strength", first.Type)
	assert.NotEmpty(t, first.Name)
	assert.NotEmpty(t, first.Difficulty)
	assert.NotEmpty(t, first.PrimaryMuscleGroup)
	assert.NotEmpty(t, first.TrackingType)
	assert.NotEmpty(t, first.Location)
	assert.NotNil(t, first.Equipment)
	assert.GreaterOrEqual(t, len(first.Instructions), 1)
	assert.GreaterOrEqual(t, len(first.Muscles), 1)
	assert.NotNil(t, first.Strength)
	assert.NotNil(t, first.Strength.Programming)

	// Validation should pass for the generated catalog
	errs := validator.Validate(catalog)
	assert.Empty(t, errs, "catalog should pass validation: %v", errs)
}
