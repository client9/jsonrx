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
