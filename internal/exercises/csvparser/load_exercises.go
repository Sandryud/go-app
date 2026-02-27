package csvparser

import (
	"fmt"
	"path/filepath"
	"strconv"

	"workout-app/data/examples"
)

// metadata from exercises.csv column (we only use searchKeywords)
type exercisesMetadata struct {
	SearchKeywords []string `json:"searchKeywords"`
}

func (p *parser) loadExercises() ([]examples.Exercise, string, error) {
	headers, rows, err := readCSV(filepath.Join(p.dir, "exercises.csv"), "exercises.csv")
	if err != nil {
		return nil, "", err
	}
	idxID, err := colIndex(headers, "id")
	if err != nil {
		return nil, "", err
	}
	idxType, err := colIndex(headers, "type")
	if err != nil {
		return nil, "", err
	}
	idxName, err := colIndex(headers, "name")
	if err != nil {
		return nil, "", err
	}
	idxDiff, err := colIndex(headers, "difficulty")
	if err != nil {
		return nil, "", err
	}
	idxMuscle, err := colIndex(headers, "primary_muscle_group")
	if err != nil {
		return nil, "", err
	}
	idxTrack, err := colIndex(headers, "tracking_type")
	if err != nil {
		return nil, "", err
	}
	idxLoc, err := colIndex(headers, "location")
	if err != nil {
		return nil, "", err
	}
	idxDesc, _ := optionalColIndex(headers, "description")
	idxCat, _ := optionalColIndex(headers, "category")
	idxMinExp, _ := optionalColIndex(headers, "minimum_experience")
	idxPop, _ := optionalColIndex(headers, "popularity_score")
	idxVerified, _ := optionalColIndex(headers, "is_verified")
	idxVersion, _ := optionalColIndex(headers, "version")
	idxMeta, _ := optionalColIndex(headers, "metadata")

	var out []examples.Exercise
	var version string
	for i, row := range rows {
		e := examples.Exercise{
			ID:                 at(row, idxID),
			Type:               at(row, idxType),
			Name:               at(row, idxName),
			Description:        at(row, idxDesc),
			Difficulty:         at(row, idxDiff),
			Category:           at(row, idxCat),
			PrimaryMuscleGroup: at(row, idxMuscle),
			TrackingType:       at(row, idxTrack),
			Location:           at(row, idxLoc),
		}
		if e.ID == "" {
			continue
		}
		if e.Difficulty == "expert" {
			e.Difficulty = "advanced"
		}
		if idxMinExp >= 0 {
			v, err := parseInt(at(row, idxMinExp))
			if err != nil {
				return nil, "", fmt.Errorf("exercises.csv row %d minimum_experience: %w", i+2, err)
			}
			e.MinExperienceMonths = v
		}
		if idxPop >= 0 {
			v, err := parseInt(at(row, idxPop))
			if err != nil {
				return nil, "", fmt.Errorf("exercises.csv row %d popularity_score: %w", i+2, err)
			}
			e.Popularity = v
		}
		if idxVerified >= 0 {
			v, err := parseBool(at(row, idxVerified))
			if err != nil {
				return nil, "", fmt.Errorf("exercises.csv row %d is_verified: %w", i+2, err)
			}
			e.IsVerified = v
		}
		if idxVersion >= 0 {
			if v := at(row, idxVersion); v != "" {
				version = v
			}
		}
		rawMeta := ""
		if idxMeta >= 0 {
			rawMeta = at(row, idxMeta)
		}
		if rawMeta != "" {
			var meta exercisesMetadata
			if err := parseJSONCell("exercises.csv row "+strconv.Itoa(i+2)+", metadata", rawMeta, &meta); err != nil {
				return nil, "", err
			}
			e.SearchKeywords = meta.SearchKeywords
		}
		out = append(out, e)
	}
	if version == "" {
		version = "2.0.0"
	}
	return out, version, nil
}
