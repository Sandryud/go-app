// Package enums holds shared allowed values for exercise CSV and catalog validation.
// Used by internal/exercises/validator and internal/exercises/csvvalidator.
package enums

// ExerciseTypes are valid values for exercise type.
var ExerciseTypes = map[string]bool{"strength": true, "cardio": true, "mobility": true, "plyometric": true}

// Difficulties are valid difficulty levels.
var Difficulties = map[string]bool{"beginner": true, "intermediate": true, "advanced": true, "expert": true}

// Locations are valid location values.
var Locations = map[string]bool{"gym": true, "home": true, "both": true}

// WarningLevels are valid warning severity levels.
var WarningLevels = map[string]bool{"critical": true, "important": true, "notice": true}

// Severities are valid contraindication severities.
var Severities = map[string]bool{"absolute": true, "relative": true}

// Relationships are valid exercise reference relationship types.
var Relationships = map[string]bool{"regression": true, "progression": true, "alternative": true}

// MuscleTypes are valid muscle activation types.
var MuscleTypes = map[string]bool{"primary": true, "secondary": true, "stabilizer": true}

// MediaTypes are valid media asset types.
var MediaTypes = map[string]bool{"image": true, "video": true}

// Forces are valid strength force types.
var Forces = map[string]bool{"push": true, "pull": true}
