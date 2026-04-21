package tojson

import "bytes"

func FromJSON5(src []byte) ([]byte, error) {
	dst := bytes.Buffer{}
	dst.Grow(len(src))
	err := FromJSON5Append(&dst, src)
	return dst.Bytes(), err
}

func FromJSON5Append(dst *bytes.Buffer, src []byte) error {
	d := decoder{
		out: dst,
	}
	return d.Translate(src)
}

func FromYAML(src []byte) ([]byte, error) {
	return yamlConvert(string(src))
}

func FromYAMLAppend(dst *bytes.Buffer, src []byte) error {
	out, err := yamlConvert(string(src))
	if err != nil {
		return err
	}
	dst.Write(out)
	return nil
}

func FromTOML(src []byte) ([]byte, error) {
	return tomlConvert(string(src))
}

func FromTOMLAppend(dst *bytes.Buffer, src []byte) error {
	out, err := tomlConvert(string(src))
	if err != nil {
		return err
	}
	dst.Write(out)
	return nil
}

// FromTOMLStreaming converts TOML to JSON using the single-pass streaming path,
// without falling back to the tree-based path on section re-entry.
func FromTOMLStreaming(src []byte) ([]byte, error) {
	return tomlConvertStreaming(string(src))
}

// FromTOMLTree converts TOML to JSON using the tree-based path directly,
// skipping the streaming attempt.
func FromTOMLTree(src []byte) ([]byte, error) {
	return tomlConvertTree(string(src))
}
