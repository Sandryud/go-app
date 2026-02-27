// Package examples содержит Go-структуры, соответствующие образцу каталога упражнений
// в data/examples/exercises.json. Используются как эталон для embed и API.
package examples

// Catalog — корневая структура JSON-каталога упражнений.
type Catalog struct {
	Meta      Meta       `json:"meta"`
	Exercises []Exercise `json:"exercises"`
}

// Meta — метаданные набора (версия, дата генерации, количество).
// Поле locale не используется: переводы slug выполняются на клиенте.
type Meta struct {
	Version     string `json:"version"`
	GeneratedAt string `json:"generated_at"`
	TotalCount  int    `json:"total_count"`
}

// Exercise — одно упражнение (общие поля + опциональный типспецифичный блок).
type Exercise struct {
	ID                 string               `json:"id"`
	Type               string               `json:"type"` // strength | cardio | mobility | plyometric
	Name               string               `json:"name"`
	Description        string               `json:"description,omitempty"`
	Difficulty         string               `json:"difficulty"`
	Category           string               `json:"category,omitempty"`
	PrimaryMuscleGroup string               `json:"primary_muscle_group"`
	TrackingType       string               `json:"tracking_type"`
	Location           string               `json:"location"`
	MinExperienceMonths int                 `json:"min_experience_months,omitempty"`
	Popularity         int                  `json:"popularity,omitempty"`
	IsVerified         bool                 `json:"is_verified,omitempty"`
	Equipment          []string             `json:"equipment,omitempty"`
	MovementPatterns   []string             `json:"movement_patterns,omitempty"`
	Purposes           []string             `json:"purposes,omitempty"`
	Tags               []string             `json:"tags,omitempty"`
	Skills             []string             `json:"skills,omitempty"`
	MeasurementUnits   []string             `json:"measurement_units,omitempty"`
	Instructions       []string             `json:"instructions,omitempty"`
	Muscles            []MuscleActivation   `json:"muscles,omitempty"`
	Media              []MediaAsset         `json:"media,omitempty"`
	Warnings           []Warning           `json:"warnings,omitempty"`
	Contraindications   []Contraindication  `json:"contraindications,omitempty"`
	Mistakes           []string            `json:"mistakes,omitempty"`
	ProgrammingNotes   []string             `json:"programming_notes,omitempty"`
	Breathing          *Breathing          `json:"breathing,omitempty"`
	References         []ExerciseReference  `json:"references,omitempty"`
	SearchKeywords     []string             `json:"search_keywords,omitempty"`
	Strength           *StrengthParams     `json:"strength,omitempty"`
	Cardio             *CardioParams       `json:"cardio,omitempty"`
	Mobility           *MobilityParams     `json:"mobility,omitempty"`
}

// MuscleActivation — вовлечение мышцы (тип, slug, уровень активации).
type MuscleActivation struct {
	Type       string `json:"type"` // primary | secondary | stabilizer
	Name       string `json:"name"`
	Activation int    `json:"activation,omitempty"`
}

// MediaAsset — медиа (фото/видео) для упражнения.
type MediaAsset struct {
	URL          string `json:"url"`
	Type         string `json:"type"` // image | video
	Alt          string `json:"alt,omitempty"`
	DurationSec  int    `json:"duration_s,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	IsPrimary    bool   `json:"is_primary,omitempty"`
	SortOrder    int    `json:"sort_order,omitempty"`
}

// Warning — предупреждение по безопасности/технике.
type Warning struct {
	Level           string   `json:"level"` // critical | important | notice
	Message         string   `json:"message"`
	BodyPart        string   `json:"body_part,omitempty"`
	Recommendations []string `json:"recommendations,omitempty"`
}

// Contraindication — противопоказание (состояние, тяжесть, альтернативы).
type Contraindication struct {
	Condition    string   `json:"condition"`
	Severity     string   `json:"severity"` // absolute | relative
	Reason       string   `json:"reason"`
	Alternatives []string `json:"alternatives,omitempty"`
}

// Breathing — блок дыхания (фазы и подсказки).
type Breathing struct {
	Eccentric   string   `json:"eccentric,omitempty"`
	Concentric  string   `json:"concentric,omitempty"`
	Description string   `json:"description,omitempty"`
	Tips        []string `json:"tips,omitempty"`
}

// ExerciseReference — связь с другим упражнением (регрессия/прогрессия/альтернатива).
type ExerciseReference struct {
	ExerciseID           string `json:"exercise_id"`
	Relationship         string `json:"relationship"` // regression | progression | alternative
	Name                 string `json:"name,omitempty"`
	EffectivenessRating  int    `json:"effectiveness_rating,omitempty"`
}

// StrengthParams — типспецифичные параметры силового упражнения.
type StrengthParams struct {
	Force        string       `json:"force,omitempty"` // push | pull
	Programming  *Programming `json:"programming,omitempty"`
}

// CardioParams — типспецифичные параметры кардио (та же схема programming при необходимости).
type CardioParams struct {
	Programming *Programming `json:"programming,omitempty"`
}

// MobilityParams — типспецифичные параметры мобильности/растяжки.
type MobilityParams struct {
	Programming *Programming `json:"programming,omitempty"`
}

// Programming — общие параметры программирования (подходы, повторения, отдых, темп, интенсивность/hold).
type Programming struct {
	SetsMin        int    `json:"sets_min,omitempty"`
	SetsMax        int    `json:"sets_max,omitempty"`
	RepsMin        int    `json:"reps_min,omitempty"`
	RepsMax        int    `json:"reps_max,omitempty"`
	RestSecMin     int    `json:"rest_sec_min,omitempty"`
	RestSecMax     int    `json:"rest_sec_max,omitempty"`
	Tempo          string `json:"tempo,omitempty"`
	IntensityPctMin int   `json:"intensity_pct_min,omitempty"`
	IntensityPctMax int   `json:"intensity_pct_max,omitempty"`
	HoldSecMin     int    `json:"hold_sec_min,omitempty"`
	HoldSecMax     int    `json:"hold_sec_max,omitempty"`
}
