// Package tojson converts JSON5, YAML, and TOML to standard JSON bytes
// without reflection or intermediate data structures.
//
// All three formats follow the same two-step pattern: convert to JSON, then
// unmarshal with the standard library using only json struct tags:
//
//	raw, err := tojson.FromYAML(src)
//	if err != nil { ... }
//	json.Unmarshal(raw, &cfg)
//
// Each format uses the primary conversion form:
//
//	tojson.FromJSON5(src []byte) ([]byte, error)
//	tojson.FromYAML(src []byte) ([]byte, error)
//	tojson.FromTOML(src []byte) ([]byte, error)
//
// Parse errors are returned as *ParseError, which carries a 1-based line
// and column number and can be inspected with errors.As:
//
//	var pe *tojson.ParseError
//	if errors.As(err, &pe) {
//	    fmt.Printf("line %d: %s\n", pe.Line, pe.Message)
//	}
package tojson
