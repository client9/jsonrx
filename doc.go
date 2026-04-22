// Package tojson converts YAML, TOML, and JSON variants to standard JSON bytes.
//
// The package is built around a simple two-step pattern: convert the source
// bytes to JSON, then unmarshal with the standard library using only json
// struct tags:
//
//	raw, err := tojson.FromYAML(src)
//	if err != nil { ... }
//	if err := json.Unmarshal(raw, &cfg); err != nil { ... }
//
// The package exposes three top-level conversion functions:
//
//	tojson.FromJSONVariant(src []byte) ([]byte, error)
//	tojson.FromYAML(src []byte) ([]byte, error)
//	tojson.FromTOML(src []byte) ([]byte, error)
//
// FromYAML intentionally supports a practical YAML subset for config files and
// frontmatter, not the full YAML specification.
//
// Parse failures are returned as *ParseError, which carries a 1-based line and
// column number and can be inspected with errors.As:
//
//	var pe *tojson.ParseError
//	if errors.As(err, &pe) {
//		fmt.Printf("line %d, column %d: %s\n", pe.Line, pe.Column, pe.Message)
//	}
package tojson
