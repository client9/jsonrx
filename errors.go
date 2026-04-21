package tojson

import "fmt"

// ParseError is returned by FromJSONVariant, FromYAML, and FromTOML when the input
// cannot be parsed. Line and Column are 1-based; Column is 0 when not available.
type ParseError struct {
	Line    int    // 1-based line number in the original input
	Column  int    // 1-based column number; 0 means not available
	Message string // description of the problem
}

func (e *ParseError) Error() string {
	if e.Column > 0 {
		return fmt.Sprintf("line %d, column %d: %s", e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}

// atLine wraps err with a 1-based line number unless it is already a ParseError.
// rawLine is a 0-based index into the original input lines.
func atLine(rawLine int, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*ParseError); ok {
		return err
	}
	return &ParseError{Line: rawLine + 1, Message: err.Error()}
}

// atLineCol wraps err with a 1-based line and column unless it is already a ParseError.
// rawLine is a 0-based index; col is a 0-based column offset.
func atLineCol(rawLine, col int, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*ParseError); ok {
		return err
	}
	return &ParseError{Line: rawLine + 1, Column: col + 1, Message: err.Error()}
}

// atToken wraps err with the 1-based line and column carried by t.
func atToken(t token, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*ParseError); ok {
		return err
	}
	return &ParseError{Line: t.row + 1, Column: t.col + 1, Message: err.Error()}
}
