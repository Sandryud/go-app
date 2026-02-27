package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"workout-app/data/examples"
)

func TestValidate_ValidCatalog(t *testing.T) {
	c := &examples.Catalog{
		Meta: examples.Meta{Version: "2.0.0", TotalCount: 1},
		Exercises: []examples.Exercise{
			{
				ID:                 "test-ex",
				Type:               "strength",
				Name:               "Test",
				Difficulty:         "intermediate",
				PrimaryMuscleGroup: "chest",
				TrackingType:       "weight-reps",
				Location:           "gym",
				Popularity:         50,
			},
		},
	}
	errs := Validate(c)
	assert.Empty(t, errs)
}

func TestValidate_InvalidDifficulty(t *testing.T) {
	c := &examples.Catalog{
		Exercises: []examples.Exercise{{
			ID:                 "x",
			Type:               "strength",
			Name:               "X",
			Difficulty:         "super",
			PrimaryMuscleGroup: "chest",
			TrackingType:       "weight-reps",
			Location:           "gym",
		}},
	}
	errs := Validate(c)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "difficulty")
	assert.Contains(t, errs[0], "beginner|intermediate|advanced|expert")
}

func TestValidate_InvalidPopularity(t *testing.T) {
	c := &examples.Catalog{
		Exercises: []examples.Exercise{{
			ID:                 "x",
			Type:               "strength",
			Name:               "X",
			Difficulty:         "beginner",
			PrimaryMuscleGroup: "chest",
			TrackingType:       "weight-reps",
			Location:           "gym",
			Popularity:         150,
		}},
	}
	errs := Validate(c)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "popularity")
}

func TestValidate_MissingRequired(t *testing.T) {
	c := &examples.Catalog{
		Exercises: []examples.Exercise{{
			ID:                 "",
			Type:               "strength",
			Name:               "X",
			Difficulty:         "beginner",
			PrimaryMuscleGroup: "chest",
			TrackingType:       "weight-reps",
			Location:           "gym",
		}},
	}
	errs := Validate(c)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "id")
}

func TestValidate_InvalidReference(t *testing.T) {
	c := &examples.Catalog{
		Exercises: []examples.Exercise{
			{
				ID:                 "a",
				Type:               "strength",
				Name:               "A",
				Difficulty:         "beginner",
				PrimaryMuscleGroup: "chest",
				TrackingType:       "weight-reps",
				Location:           "gym",
				References: []examples.ExerciseReference{
					{ExerciseID: "nonexistent", Relationship: "progression"},
				},
			},
		},
	}
	errs := Validate(c)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "nonexistent")
}

func TestValidate_NilCatalog(t *testing.T) {
	errs := Validate(nil)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "nil")
}

func TestValidate_DuplicateExerciseID(t *testing.T) {
	c := &examples.Catalog{
		Exercises: []examples.Exercise{
			{
				ID:                 "dup-id",
				Type:               "strength",
				Name:               "First",
				Difficulty:         "beginner",
				PrimaryMuscleGroup: "chest",
				TrackingType:       "weight-reps",
				Location:           "gym",
			},
			{
				ID:                 "dup-id",
				Type:               "strength",
				Name:               "Second",
				Difficulty:         "beginner",
				PrimaryMuscleGroup: "back",
				TrackingType:       "weight-reps",
				Location:           "gym",
			},
		},
	}
	errs := Validate(c)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "duplicate exercise id")
	assert.Contains(t, errs[0], "dup-id")
}

func TestValidationError_String(t *testing.T) {
	e := ValidationError{ExerciseID: "ex-1", Field: "popularity", Message: "must be 0-100"}
	s := e.String()
	assert.Contains(t, s, "ex-1")
	assert.Contains(t, s, "popularity")
	assert.Contains(t, s, "must be 0-100")
}
