package jsonrx

import "bytes"

func Decode(src []byte) ([]byte, error) {
	dst := bytes.Buffer{}
	dst.Grow(len(src))
	err := DecodeAppend(&dst, src)
	return dst.Bytes(), err
}

func DecodeAppend(dst *bytes.Buffer, src []byte) error {
	d := decoder{
		out: dst,
	}
	return d.Translate(src)
}
