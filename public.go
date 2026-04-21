package tojson

import "bytes"

// FromJSON5 converts JSON5/HuJSON/JWCC/JSONC/HanSON to standard JSON.
// It handles trailing/leading commas, line and block comments, unquoted keys,
// single-quoted and backtick strings, hex literals, and non-finite numbers.
func FromJSON5(src []byte) ([]byte, error) {
	dst := bytes.Buffer{}
	dst.Grow(len(src))
	d := decoder{
		out: &dst,
	}
	err := d.Translate(src)
	return dst.Bytes(), err
}

// FromYAML converts a YAML subset to standard JSON.
// The output can be passed directly to encoding/json.Unmarshal using only json struct tags.
// Anchors/aliases, tags, and complex keys are not supported.
func FromYAML(src []byte) ([]byte, error) {
	return yamlConvert(src)
}

// FromTOML converts TOML to standard JSON.
// The output can be passed directly to encoding/json.Unmarshal using only json struct tags.
func FromTOML(src []byte) ([]byte, error) {
	return tomlConvert(src)
}

// fromTOMLStreaming converts TOML to JSON using the single-pass streaming path,
// without falling back to the tree-based path on section re-entry.
func fromTOMLStreaming(src []byte) ([]byte, error) {
	return tomlConvertStreaming(src)
}

// fromTOMLTree converts TOML to JSON using the tree-based path directly,
// skipping the streaming attempt.
func fromTOMLTree(src []byte) ([]byte, error) {
	return tomlConvertTree(src)
}
