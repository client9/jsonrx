package jsonrx

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
