package csvparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBool(t *testing.T) {
	tests := []struct {
		s    string
		want bool
		ok   bool
	}{
		{"true", true, true},
		{"false", false, true},
		{"1", true, true},
		{"0", false, true},
		{"", false, true},
		{"yes", true, true},
		{"no", false, true},
		{"invalid", false, false},
	}
	for _, tt := range tests {
		got, err := parseBool(tt.s)
		if tt.ok {
			require.NoError(t, err)
			assert.Equal(t, tt.want, got, "parseBool(%q)", tt.s)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		s    string
		want int
		ok   bool
	}{
		{"0", 0, true},
		{"42", 42, true},
		{"", 0, true},
		{"-1", -1, true},
		{"abc", 0, false},
	}
	for _, tt := range tests {
		got, err := parseInt(tt.s)
		if tt.ok {
			require.NoError(t, err)
			assert.Equal(t, tt.want, got, "parseInt(%q)", tt.s)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestParseJSONCell_Programming(t *testing.T) {
	raw := `{"sets":{"min":3,"max":5},"reps":{"min":6,"max":12},"rest":{"min":120,"max":180},"tempo":"3-1-1-0","intensity":{"min":70,"max":85}}`
	var p programmingJSON
	err := parseJSONCell("test", raw, &p)
	require.NoError(t, err)
	assert.NotNil(t, p.Sets)
	assert.Equal(t, 3, p.Sets.Min)
	assert.Equal(t, 5, p.Sets.Max)
	assert.Equal(t, "3-1-1-0", p.Tempo)
	assert.NotNil(t, p.Intensity)
	assert.Equal(t, 70, p.Intensity.Min)
	assert.Equal(t, 85, p.Intensity.Max)
}

func TestParseJSONCell_Recommendations(t *testing.T) {
	raw := `{"recommendations":["Tip one","Tip two"]}`
	var r struct {
		Recommendations []string `json:"recommendations"`
	}
	err := parseJSONCell("warnings", raw, &r)
	require.NoError(t, err)
	assert.Equal(t, []string{"Tip one", "Tip two"}, r.Recommendations)
}

func TestParseJSONCell_Alternatives(t *testing.T) {
	raw := `{"alternatives":["ex-1","ex-2"]}`
	var a struct {
		Alternatives []string `json:"alternatives"`
	}
	err := parseJSONCell("contraindications", raw, &a)
	require.NoError(t, err)
	assert.Equal(t, []string{"ex-1", "ex-2"}, a.Alternatives)
}

func TestParseJSONCell_Invalid(t *testing.T) {
	var v map[string]interface{}
	err := parseJSONCell("file.csv row 2, col", "{invalid json", &v)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file.csv")
}

func TestColIndex(t *testing.T) {
	headers := []string{"id", "name", "type"}
	i, err := colIndex(headers, "name")
	require.NoError(t, err)
	assert.Equal(t, 1, i)
	_, err = colIndex(headers, "missing")
	assert.Error(t, err)
}

func TestAt(t *testing.T) {
	row := []string{"a", "b", "c"}
	assert.Equal(t, "b", at(row, 1))
	assert.Equal(t, "", at(row, -1))
	assert.Equal(t, "", at(row, 10))
}

func TestProgrammingToStruct(t *testing.T) {
	prog := programmingJSON{
		Sets:      &minMax{3, 5},
		Reps:      &minMax{6, 12},
		Rest:      &minMax{120, 180},
		Tempo:     "3-1-1-0",
		Intensity: &minMax{70, 85},
	}
	p := programmingToStruct(prog)
	require.NotNil(t, p)
	assert.Equal(t, 3, p.SetsMin)
	assert.Equal(t, 5, p.SetsMax)
	assert.Equal(t, 6, p.RepsMin)
	assert.Equal(t, 12, p.RepsMax)
	assert.Equal(t, 120, p.RestSecMin)
	assert.Equal(t, 180, p.RestSecMax)
	assert.Equal(t, "3-1-1-0", p.Tempo)
	assert.Equal(t, 70, p.IntensityPctMin)
	assert.Equal(t, 85, p.IntensityPctMax)
}

func TestProgrammingToStruct_WithHold(t *testing.T) {
	prog := programmingJSON{Hold: &minMax{30, 60}}
	p := programmingToStruct(prog)
	require.NotNil(t, p)
	assert.Equal(t, 30, p.HoldSecMin)
	assert.Equal(t, 60, p.HoldSecMax)
}
