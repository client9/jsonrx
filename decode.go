package jsonrx

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"
)

// from Javascript
// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Number/MIN_SAFE_INTEGER
//
//	Number.MaxSafeInteger
//	Number.MinSafeInteger
var bareInfinity = []byte("Infinity")
var maxSafeInteger = []byte("9007199254740991")
var minSafeInteger = []byte("-9007199254740992")

type decoder struct {
	tok   *tokenizer
	out   *bytes.Buffer
	stack []byte
	next  stateFunction
}

type stateFunction func(d *decoder, t token) error

func (d *decoder) Translate(src []byte) error {
	d.tok = newTokenizer(src)
	d.next = stateValue

	for {
		t, err := d.tok.Next()
		if err == io.EOF {
			if len(d.stack) == 0 {
				return nil
			}
			return fmt.Errorf("Got EOF midway")
		}
		if err != nil {
			return err
		}

		// ignore comments
		if t.kind == 'c' {
			continue
		}
		err = d.next(d, t)

		if err != nil {
			return err
		}
	}
}

func stateValue(d *decoder, t token) error {
	switch t.kind {
	case leftBrace:
		return stateObjectStart(d, t)
	case leftBracket:
		return stateArrayStart(d, t)
	case 's':
		writeString(d.out, t.value)
		d.next = stateObjectAfterValue
	case '0':
		writeInt(d.out, t.value)
		d.next = stateObjectAfterValue
	case '1':
		writeFloat(d.out, t.value)
		d.next = stateObjectAfterValue
	case '2':
		writeHex(d.out, t.value)
		d.next = stateObjectAfterValue
	case 'w':
		bareword(d.out, t.value)
		d.next = stateObjectAfterValue
	default:
		return fmt.Errorf("Unknown token for value")
	}
	return nil
}

func stateObjectStart(d *decoder, t token) error {
	d.out.WriteByte('{')
	d.stack = append(d.stack, '{')
	d.next = stateObjectAfterStart
	return nil
}

func stateObjectAfterStart(d *decoder, t token) error {
	switch t.kind {
	case '}':
		return stateObjectEnd(d, t)
	case ',': // degenerate case
		// ignore comma and reparse
		d.next = stateObjectAfterStart
		return nil
	}
	return stateObjectKey(d, t)
}

func stateObjectKey(d *decoder, t token) error {
	switch t.kind {
	case 's':
		writeString(d.out, t.value)
		d.next = stateObjectAfterKey
	case 'w', '0', '1', '2':
		// whatever it is, it's always quoted
		writeQuoted(d.out, t.value)
		d.next = stateObjectAfterKey
	default:
		return fmt.Errorf("Invalid token at Object key :%s", t)
	}
	return nil
}

func stateObjectAfterKey(d *decoder, t token) error {
	if t.kind == ':' {
		d.out.Write(t.value)
		d.next = stateObjectValue
		return nil
	}

	return fmt.Errorf("Invalid token after Object Key")
}

func stateObjectValue(d *decoder, t token) error {
	switch t.kind {
	case 's':
		writeString(d.out, t.value)
		d.next = stateObjectAfterValue
	case 'w':
		bareword(d.out, t.value)
		d.next = stateObjectAfterValue
	case '0':
		writeInt(d.out, t.value)
		d.next = stateObjectAfterValue
	case '1':
		writeFloat(d.out, t.value)
		d.next = stateObjectAfterValue
	case '2':
		writeHex(d.out, t.value)
		d.next = stateObjectAfterValue
	case '{':
		return stateObjectStart(d, t)
	case '[':
		return stateArrayStart(d, t)
	default:
		return fmt.Errorf("Unknown token for object value - %s", t)
	}
	return nil
}
func stateObjectAfterValue(d *decoder, t token) error {
	switch t.kind {
	case '}':
		return stateObjectEnd(d, t)
	case ',':
		return stateComma(d, t)
		// MIDDLE COMMA
	case 'w', 's', '0', '1', '2':
		// e.g. { "key": 1 "key2": 2 }  ==> { "key": 1, "key2": 2 }
		d.out.WriteByte(',')
		return stateObjectKey(d, t)
	default:
		return fmt.Errorf("Unknown token after object value - %s", t)
	}
}

func stateComma(d *decoder, t token) error {
	// check if next token is "}"

	t2, err := d.tok.Next()
	if err != nil {
		return err
	}

	if t2.kind == 'c' {
		// if comment, reparse
		return stateComma(d, t)
	}
	if t2.kind == '}' {
		// Skip writing comma
		return stateObjectEnd(d, t2)
	}
	if t2.kind == ']' {
		return stateArrayEnd(d, t2)
	}

	d.out.Write(t.value)

	if d.stack[len(d.stack)-1] == '{' {
		// write comma, and expect a key
		return stateObjectKey(d, t2)
	}

	// it's an array value
	return stateArrayValue(d, t2)
}

func stateObjectEnd(d *decoder, t token) error {
	if len(d.stack) == 0 || d.stack[len(d.stack)-1] != '{' {
		return fmt.Errorf("Unmatched object end, level=%d, stack=%q", len(d.stack), string(d.stack))
	}
	d.out.WriteByte('}')
	d.stack = d.stack[:len(d.stack)-1]
	d.next = stateAfterContainer
	return nil
}

func stateAfterContainer(d *decoder, t token) error {

	switch t.kind {
	case rightBrace:
		return stateObjectEnd(d, t)
	case rightBracket:
		return stateArrayEnd(d, t)
	case ',':
		return stateComma(d, t)

	// MIDDLE COMMA
	case leftBrace, leftBracket, 'w', 's', '0', '1', '2':
		d.out.WriteByte(',')
		if d.stack[len(d.stack)-1] == leftBrace {
			// write comma, and expect a key
			return stateObjectKey(d, t)
		} else {
			// it's an array value
			return stateArrayValue(d, t)
		}

	default:
		return fmt.Errorf("unknown token after end of object or array - %s", t)
	}
}

func stateArrayStart(d *decoder, t token) error {
	d.out.WriteByte('[')
	d.stack = append(d.stack, '[')
	d.next = stateArrayAfterStart
	return nil
}

func stateArrayAfterStart(d *decoder, t token) error {
	switch t.kind {
	case ']':
		return stateArrayEnd(d, t)
	case ',': // degenerate case
		// ignore comma and reparse
		d.next = stateArrayAfterStart
		return nil
	}
	return stateArrayValue(d, t)
}

func stateArrayEnd(d *decoder, t token) error {
	if len(d.stack) == 0 || d.stack[len(d.stack)-1] != '[' {
		return fmt.Errorf("Unmatched array end")
	}
	d.out.WriteByte(']')
	d.stack = d.stack[:len(d.stack)-1]
	d.next = stateAfterContainer
	return nil
}

func stateArrayValue(d *decoder, t token) error {
	switch t.kind {
	case 's':
		writeString(d.out, t.value)
		d.next = stateArrayAfterValue
	case 'w':
		bareword(d.out, t.value)
		d.next = stateArrayAfterValue
	case '0':
		writeInt(d.out, t.value)
		d.next = stateArrayAfterValue
	case '1':
		writeFloat(d.out, t.value)
		d.next = stateArrayAfterValue
	case '2':
		writeHex(d.out, t.value)
		d.next = stateArrayAfterValue
	case '{':
		return stateObjectStart(d, t)
	case '[':
		return stateArrayStart(d, t)
	default:
		return fmt.Errorf("Unknown token for array value - %s", t)
	}
	return nil
}

func stateArrayAfterValue(d *decoder, t token) error {
	switch t.kind {
	case ']':
		return stateArrayEnd(d, t)
	case ',':
		return stateComma(d, t)

		// MIDDLE COMMA
	case leftBrace, leftBracket, 'w', 's', '0', '1', '2':
		// e.g. [ 1 2 3 ] ==> [ 1,2,3 ]
		//      [ "foo" "bar" ] ==> [ "foo", "bar" ]
		d.out.WriteByte(',')
		return stateArrayValue(d, t)
	}
	return fmt.Errorf("Unknown token after array value - %s", t)
}

func isNull(b []byte) bool {
	if len(b) != 4 {
		return false
	}
	return b[0] == 'n' &&
		b[1] == 'u' &&
		b[2] == 'l' &&
		b[3] == 'l'
}
func isTrue(b []byte) bool {
	if len(b) != 4 {
		return false
	}
	return b[0] == 't' &&
		b[1] == 'r' &&
		b[2] == 'u' &&
		b[3] == 'e'
}
func isFalse(b []byte) bool {
	if len(b) != 5 {
		return false
	}
	return b[0] == 'f' &&
		b[1] == 'a' &&
		b[2] == 'l' &&
		b[3] == 's' &&
		b[4] == 'e'
}

func writeInt(out *bytes.Buffer, b0 []byte) {

	// assert -- should never happen
	if len(b0) == 0 {
		return
	}

	b := b0
	//leading := false
	if b[0] == '-' || b[0] == '+' {
		b = b[1:]
	}
	val := uint64(0)
	overflow := false
	notint := false
	for _, c := range b {
		// here for safety
		if '0' <= c && c <= '9' {
			d := uint64(c - '0')
			val = val*10 + d
			if val > (1 << 54) {
				overflow = true
				break
			}
		} else {
			notint = true
			break
		}
	}

	if overflow {
		writeFloat(out, b0)
		return
	}
	if notint {
		bareword(out, b0)
		return
	}

	b = b0
	// if we got here, we have a valid integer
	// just check if we start with a '+' which is un-needed
	if b[0] == '+' {
		b = b[1:]
	}

	// trim off leading zeros
	for len(b) > 0 && b[0] == '0' {
		b = b[1:]
	}
	if len(b) == 0 {
		// "+00000" or "0000"
		out.WriteByte('0')
		return
	}

	out.Write(b)
}

// Unoptimized since it's a rare feature
func writeHex(out *bytes.Buffer, b []byte) {

	// need to slice off initial "0x" or "0X"
	s := string(b[2:])
	numInt, err := strconv.ParseInt(s, 16, 64)
	if err == nil {
		out.WriteString(strconv.FormatInt(numInt, 10))
		return
	}
	// TODO - Overflow
	bareword(out, b)
	return
}

func writeFloat(out *bytes.Buffer, b []byte) {
	if isValidNumber(b) {
		out.Write(b)
		return
	}

	// TODO
	// https://cs.opensource.google/go/go/+/refs/tags/go1.24.3:src/encoding/json/encode.go
	num, err := strconv.ParseFloat(string(b), 64)
	if err == nil {
		out.WriteString(fmt.Sprintf("%g", num))
		return
	}

	// TODO - overflow
	out.Write(b)
	return
}

func bareword(out *bytes.Buffer, b []byte) {
	if isNull(b) || isTrue(b) || isFalse(b) {
		out.Write(b)
		return
	}

	if bytes.Equal(bareInfinity, b) || (b[0] == '+' && bytes.Equal(bareInfinity, b[1:])) {
		out.Write(maxSafeInteger)
		return
	}
	if b[0] == '-' && bytes.Equal(bareInfinity, b[1:]) {
		out.Write(minSafeInteger)
		return
	}

	// TODO Nan

	// something else
	out.Write(b)
	return
}

// writeString takes an "quoted string with escapes" and converts to a JSON-spec string.
// it needs to handle
//
//   - single quote strings
//   - double quote strings
//   - backtick quote strings
//
// all with a variety of escape sequences.
func writeString(out *bytes.Buffer, src []byte) {
	// get quote type
	qchar := src[0]

	// strip off quotes
	src = src[1 : len(src)-1]

	// do we need to decode anything?
	hasEscape := false
	for _, b := range src {
		if b < utf8.RuneSelf && !safeSet[b] {
			hasEscape = true
			break
		}
	}

	if !hasEscape {
		out.WriteByte('"')
		out.Write(src)
		out.WriteByte('"')
		return
	}

	if qchar == backQuote {
		// TBD: ERROR -- \` needs to be unescaped.
		//
		// no need to unescape first -- directly encode
		writeQuoted(out, src)
		return
	}

	buf := out.AvailableBuffer()
	buf = appendRecodeString(buf, src)
	out.Write(buf)
	return
}

func writeQuoted(out *bytes.Buffer, src []byte) {
	/*
		hasEscape := false
		for _, b := range src {
			if b < utf8.RuneSelf && !safeSet[b] {
				hasEscape = true
				break
			}
		}

		if !hasEscape {
			out.WriteByte('"')
			out.Write(src)
			out.WriteByte('"')
			return
		}
	*/
	buf := out.AvailableBuffer()
	buf = appendString(buf, src)
	out.Write(buf)
}
