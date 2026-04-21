package tojson

import "fmt"

// ParseError is returned by FromJSON5, FromYAML, and FromTOML when the input
// cannot be parsed. Line is 1-based.
type ParseError struct {
	Line int    // 1-based line number in the original input
	Msg  string // description of the problem
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Msg)
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
	return &ParseError{Line: rawLine + 1, Msg: err.Error()}
}

// atToken wraps err with the 1-based line number carried by t.
func atToken(t token, err error) error {
	return atLine(t.row, err)
}
