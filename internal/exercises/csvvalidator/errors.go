// Package csvvalidator validates exercise CSV files (structure, types, enums, cross-references).
package csvvalidator

import "fmt"

// Error holds file, row (1-based), column, and message for a single validation error.
type Error struct {
	File    string
	Row     int
	Column  string
	Message string
}

// Error implements the error interface.
func (e Error) Error() string {
	return e.String()
}

func (e Error) String() string {
	if e.Column != "" {
		return fmt.Sprintf("%s row %d column %q: %s", e.File, e.Row, e.Column, e.Message)
	}
	return fmt.Sprintf("%s row %d: %s", e.File, e.Row, e.Message)
}
