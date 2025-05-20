package jsonrx

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

const leftBrace = '{'
const rightBrace = '}'
const leftBracket = '['
const rightBracket = ']'
const comma = ','
const colon = ':'
const singleQuote = '\''
const doubleQuote = '"'
const backQuote = '`'
const backslash = '\\'
const newline = '\n'
const slash = '/'
const aster = '*'

type token struct {
	kind  byte
	value []byte
	row   int
	col   int
}

func (t token) String() string {
	kind := string([]byte{t.kind})
	return fmt.Sprintf("type %s: %s @ %d:%d", kind, string(t.value), t.row, t.col)
}

type jsonRx struct {
	row  int
	col  int
	data []byte

	stack []byte
}

func newJsonRx(b []byte) *jsonRx {
	return &jsonRx{data: b}
}

func (tx *jsonRx) Next() (token, error) {
	if len(tx.data) == 0 {
		return token{}, io.EOF
	}
	//fmt.Printf("Left: %q\n", string(tx.data))
	for i, b := range tx.data {
		switch b {
		// single char tokens
		case leftBrace, rightBrace, leftBracket, rightBracket, comma, colon:
			tx.data = tx.data[i:]
			t := token{
				kind:  b,
				value: tx.data[0:1],
				row:   tx.row,
				col:   tx.col,
			}
			tx.col += 1
			tx.data = tx.data[1:]
			return t, nil
		case ' ', '\t':
			tx.col += 1
			continue
		case newline:
			tx.row += 1
			tx.col = 0
			continue
		case singleQuote, doubleQuote, backQuote:
			tx.data = tx.data[i:]
			return tx.string()
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '+', '-', '.':
			tx.data = tx.data[i:]
			return tx.number()
		case slash:
			tx.data = tx.data[i:]
			return tx.comment()
		case '#':
			tx.data = tx.data[i:]
			return tx.comment()
		default:
			tx.data = tx.data[i:]
			return tx.bareword()
		}
	}
	return token{}, io.EOF
}

func (tx *jsonRx) string() (token, error) {
	qchar := tx.data[0]

	skip := false
	for i, b := range tx.data[1:] {
		switch b {
		case qchar:
			if skip {
				skip = false
				continue
			}
			// +1 for the first char we skipped in loop
			// +1 for keeping last quote
			i += 2
			t := token{
				kind:  's',
				value: tx.data[:i],
				row:   tx.row,
				col:   tx.col,
			}
			tx.data = tx.data[i:]
			return t, nil
		case backslash:
			skip = true
		case newline:
			if !skip && qchar == backQuote {
				tx.row += 1
				continue
			}

			if !skip {
				return token{}, fmt.Errorf("Unescaped newline in string")
			}
			skip = false
			tx.row += 1
			// remap from \[newline] to \n
			tx.data[i] = 'n'
		default:
			if skip {
				skip = false
			}
		}
	}
	// always an error
	return token{}, fmt.Errorf("Quoted string fell off edge")
}

func (tx *jsonRx) comment() (token, error) {
	if tx.data[0] == '#' {
		return tx.commentSingle()
	}
	if len(tx.data) > 1 {
		switch tx.data[1] {
		case slash:
			return tx.commentSingle()
		case aster:
			return tx.commentMulti()
		}
	}
	// reparse as bareword
	return tx.bareword()
}

func (tx *jsonRx) commentMulti() (token, error) {
	endAster := false
	row := tx.row
	col := tx.col
	var i int
	for i, b := range tx.data[2:] {
		switch b {
		case newline:
			tx.row += 1
			tx.col = 0
		case aster:
			tx.col += 1
			endAster = true
		case slash:
			if endAster {
				i += 3
				t := token{
					kind:  'c',
					value: tx.data[:i],
					row:   row,
					col:   col,
				}
				tx.data = tx.data[i:]
				return t, nil
			}
		default:
			endAster = false
			tx.col += 1
		}
	}

	// multi-line comment wasn't closed

	t := token{
		kind:  'c',
		value: tx.data[:i],
		row:   row,
		col:   col,
	}
	tx.data = tx.data[i:]
	return t, nil
}
func (tx *jsonRx) commentSingle() (token, error) {
	t := token{
		kind:  'c',
		value: tx.data,
		row:   tx.row,
		col:   tx.col,
	}
	for i, b := range tx.data {
		if b == newline {
			t.value = tx.data[:i]
			tx.data = tx.data[i:]
			return t, nil
		}
	}
	tx.data = []byte{}
	return t, nil
}

func (tx *jsonRx) hexnumber() (token, error) {

	kind := byte('2')

	// [2:] is safe since checked already
	for i, b := range tx.data[2:] {
		switch b {

		case leftBrace, rightBrace, leftBracket, rightBracket, colon, comma, ' ', '\t', '\n':

			t := token{
				kind:  kind,
				value: tx.data[:i+2],
				row:   tx.row,
				col:   tx.col,
			}
			tx.col += i
			tx.data = tx.data[i+2:]
			return t, nil
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f', 'A', 'B', 'C', 'D', 'E', 'F':
			// keep going

		default:
			// non hex characters found
			// treat as regular bareword
			kind = 'w'
		}
	}
	// fell off the edge
	t := token{
		kind:  kind,
		value: tx.data,
		row:   tx.row,
		col:   tx.col,
	}
	tx.col += len(tx.data)
	tx.data = []byte{}
	return t, nil
}
func (tx *jsonRx) number() (token, error) {

	// check if starts with "0x"
	if len(tx.data) > 2 && tx.data[0] == '0' && (tx.data[1] == 'x' || tx.data[1] == 'X') {
		return tx.hexnumber()
	}
	kind := byte('0') // integer
	for i, b := range tx.data {
		switch b {

		case leftBrace, rightBrace, leftBracket, rightBracket, colon, comma, ' ', '\t', '\n':
			t := token{
				kind:  kind,
				value: tx.data[:i],
				row:   tx.row,
				col:   tx.col,
			}
			tx.col += i
			tx.data = tx.data[i:]
			return t, nil
		case '+', '-':
			// TBD keep going
			// exponential
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			//
		case '.', 'e':
			// it's definitely not an integer
			// if already marked as 'w', keep as 'w'
			if kind == '0' {
				kind = '1'
			}
		default:
			// this isn't a number
			kind = 'w'
		}
	}
	// fell off the edge
	t := token{
		kind:  kind,
		value: tx.data,
		row:   tx.row,
		col:   tx.col,
	}
	tx.col += len(tx.data)
	tx.data = []byte{}
	return t, nil
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
func (tx *jsonRx) bareword() (token, error) {
	for i, b := range tx.data {
		switch b {
		case leftBrace, rightBrace, leftBracket, rightBracket, colon, comma, ' ', '\t', '\n':
			t := token{
				kind:  'w',
				value: tx.data[0:i],
				row:   tx.row,
				col:   tx.col,
			}
			tx.data = tx.data[i:]
			tx.col += i
			return t, nil
		default:
		}
	}
	// fell off the edge
	t := token{
		kind:  'w',
		value: tx.data,
		row:   tx.row,
		col:   tx.col,
	}
	tx.col += len(tx.data)
	tx.data = []byte{}
	return t, nil
}

type State int

const (
	StateZero State = iota
	stateValue
	StateObjectStart
	StateObjectAfterStart
	StateObjectEnd
	StateObjectKey
	StateObjectAfterKey
	StateObjectValue
	StateObjectAfterValue
	StateArrayStart
	StateArrayAfterStart
	StateArrayAfterValue
	StateArrayEnd
	StateAfterContainer
)

func (rx *jsonRx) Translate(out *bytes.Buffer) error {
	rx.stack = []byte{}
	var err error
	var t token
	state := stateValue
	for {
		t, err = rx.Next()
		if err == io.EOF {
			if len(rx.stack) == 0 {
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

		//fmt.Printf("STATE: %d with token %s\n", state, t)
		switch state {
		case stateValue:
			state, err = rx.stateValue(t, out)
			if err != nil {
				return err
			}
			if len(rx.stack) == 0 {
				// we are done.. single word.
				return nil
			}
		case StateObjectStart:
			state, err = rx.stateObjectStart(t, out)
		case StateObjectAfterStart:
			state, err = rx.stateObjectAfterStart(t, out)
		case StateObjectKey:
			state, err = rx.stateObjectKey(t, out)
		case StateObjectAfterKey:
			state, err = rx.stateObjectAfterKey(t, out)
		case StateObjectValue:
			state, err = rx.stateObjectValue(t, out)
		case StateObjectAfterValue:
			state, err = rx.stateObjectAfterValue(t, out)
		case StateObjectEnd:
			state, err = rx.stateObjectEnd(t, out)
		case StateArrayStart:
			state, err = rx.stateArrayStart(t, out)
		case StateArrayAfterStart:
			state, err = rx.stateArrayAfterStart(t, out)
		case StateArrayAfterValue:
			state, err = rx.stateArrayAfterValue(t, out)
		case StateArrayEnd:
			state, err = rx.stateArrayEnd(t, out)
		case StateAfterContainer:
			state, err = rx.stateAfterContainer(t, out)
		default:
			err = fmt.Errorf("Unknown state")
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (rx *jsonRx) stateValue(t token, out *bytes.Buffer) (State, error) {
	switch t.kind {
	case leftBrace:
		return rx.stateObjectStart(t, out)
	case leftBracket:
		return rx.stateArrayStart(t, out)
	case 's':
		writeString(out, t.value)
		return StateObjectAfterValue, nil
	case '0':
		out.Write(writeInt(t.value))
		return StateObjectAfterValue, nil
	case '1':
		out.Write(writeFloat(t.value))
		return StateObjectAfterValue, nil
	case '2':
		out.Write(writeHex(t.value))
		return StateObjectAfterValue, nil
	case 'w':
		out.Write(bareword(t.value))
		return StateObjectAfterValue, nil
	}
	return StateZero, fmt.Errorf("Unknown token for value")
}
func (rx *jsonRx) stateObjectStart(t token, out *bytes.Buffer) (State, error) {
	out.WriteByte('{')
	rx.stack = append(rx.stack, '{')
	return StateObjectAfterStart, nil
}

func (rx *jsonRx) stateObjectAfterStart(t token, out *bytes.Buffer) (State, error) {

	switch t.kind {
	case '}':
		return rx.stateObjectEnd(t, out)
	case ',': // degenerate case
		// ignore comma and reparse
		return StateObjectAfterStart, nil
	}
	return rx.stateObjectKey(t, out)

}
func (rx *jsonRx) stateObjectKey(t token, out *bytes.Buffer) (State, error) {

	switch t.kind {
	case 's':
		writeString(out, t.value)
		return StateObjectAfterKey, nil
	case 'w', '0', '1', '2':
		// whatever it is, it's always quoted
		writeQuoted(t.value, out)
		return StateObjectAfterKey, nil
	}
	return StateZero, fmt.Errorf("Invalid token at Object Key: %s", t.String())

}

func (rx *jsonRx) stateObjectAfterKey(t token, out *bytes.Buffer) (State, error) {

	if t.kind == ':' {
		out.Write(t.value)
		return StateObjectValue, nil
	}

	return StateZero, fmt.Errorf("Invalid token after Object Key")
}

func (rx *jsonRx) stateObjectValue(t token, out *bytes.Buffer) (State, error) {
	switch t.kind {
	case 's':
		writeString(out, t.value)
		return StateObjectAfterValue, nil
	case 'w':
		out.Write(bareword(t.value))
		return StateObjectAfterValue, nil
	case '0':
		out.Write(writeInt(t.value))
		return StateObjectAfterValue, nil
	case '1':
		out.Write(writeFloat(t.value))
		return StateObjectAfterValue, nil
	case '2':
		out.Write(writeHex(t.value))
		return StateObjectAfterValue, nil
	case '{':
		return rx.stateObjectStart(t, out)
	case '[':
		return rx.stateArrayStart(t, out)
	}
	return StateZero, fmt.Errorf("Unknown token for object value - %s", t)
}
func (rx *jsonRx) stateObjectAfterValue(t token, out *bytes.Buffer) (State, error) {
	switch t.kind {
	case '}':
		return rx.stateObjectEnd(t, out)
	case ',':
		return rx.stateComma(t, out)
	}
	return StateZero, fmt.Errorf("Unknown token after object value - %s", t)
}
func (rx *jsonRx) stateComma(t token, out *bytes.Buffer) (State, error) {
	// check if next token is "}"

	t2, err := rx.Next()
	if err != nil {
		return StateZero, err
	}

	if t2.kind == 'c' {
		// if comment, reparse
		return rx.stateComma(t, out)
	}
	if t2.kind == '}' {
		// Skip writing comma
		return rx.stateObjectEnd(t2, out)
	}
	if t2.kind == ']' {
		return rx.stateArrayEnd(t2, out)
	}

	out.Write(t.value)

	if rx.stack[len(rx.stack)-1] == '{' {
		// write comma, and expect a key
		return rx.stateObjectKey(t2, out)
	}

	// it's an array value
	return rx.stateArrayValue(t2, out)
}
func (rx *jsonRx) stateObjectEnd(t token, out *bytes.Buffer) (State, error) {
	stack := rx.stack
	if len(stack) == 0 || stack[len(stack)-1] != '{' {
		return StateZero, fmt.Errorf("Unmatched object end, level=%d, stack=%q", len(stack), string(stack))
	}
	out.WriteByte('}')
	rx.stack = stack[:len(stack)-1]
	return StateAfterContainer, nil
}

func (rx *jsonRx) stateAfterContainer(t token, out *bytes.Buffer) (State, error) {

	switch t.kind {
	case '}':
		return rx.stateObjectEnd(t, out)
	case ']':
		return rx.stateArrayEnd(t, out)
	case ',':
		return rx.stateComma(t, out)
	}
	return StateZero, fmt.Errorf("unknown token after end of array or object")
}
func (rx *jsonRx) stateArrayStart(t token, out *bytes.Buffer) (State, error) {
	out.WriteByte('[')
	rx.stack = append(rx.stack, '[')
	return StateArrayAfterStart, nil
}
func (rx *jsonRx) stateArrayAfterStart(t token, out *bytes.Buffer) (State, error) {

	switch t.kind {
	case ']':
		return rx.stateArrayEnd(t, out)
	case ',': // degenerate case
		// ignore comma and reparse
		return StateArrayAfterStart, nil
	}
	return rx.stateArrayValue(t, out)
}
func (rx *jsonRx) stateArrayEnd(t token, out *bytes.Buffer) (State, error) {
	stack := rx.stack
	if len(stack) == 0 || stack[len(stack)-1] != '[' {
		return StateZero, fmt.Errorf("Unmatched array end")
	}
	out.WriteByte(']')
	rx.stack = stack[:len(stack)-1]
	return StateAfterContainer, nil
}
func (rx *jsonRx) stateArrayValue(t token, out *bytes.Buffer) (State, error) {
	switch t.kind {
	case 's':
		writeString(out, t.value))
		return StateArrayAfterValue, nil
	case 'w':
		out.Write(bareword(t.value))
		return StateArrayAfterValue, nil
	case '0':
		out.Write(writeInt(t.value))
		return StateArrayAfterValue, nil
	case '1':
		out.Write(writeFloat(t.value))
		return StateArrayAfterValue, nil
	case '2':
		out.Write(writeHex(t.value))
		return StateArrayAfterValue, nil
	case '{':
		return rx.stateObjectStart(t, out)
	case '[':
		return rx.stateArrayStart(t, out)
	}
	return StateZero, fmt.Errorf("Unknown token for array value - %s", t)
}

func (rx *jsonRx) stateArrayAfterValue(t token, out *bytes.Buffer) (State, error) {
	switch t.kind {
	case ']':
		return rx.stateArrayEnd(t, out)
	case ',':
		return rx.stateComma(t, out)
	}
	return StateZero, fmt.Errorf("Unknown token after array value - %s", t)
}

func writeInt(b0 []byte) []byte {

	// asset -- should never happen
	if len(b0) == 0 {
		return nil
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
		return writeFloat(b0)
	}
	if notint {
		return bareword(b0)
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
		return []byte{'0'}
	}

	return b
}

func writeInt2(b []byte) []byte {
	s := string(b)
	numInt, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		// OVERFLOW
		if numInt >= (int64(1) << 53) {
			return writeFloat(b)
			//return []byte("12345")
		}
		if numInt <= -(int64(1) << 53) {
			return writeFloat(b)
			//return []byte("-12345")
		}
		if false {
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
				return []byte{'0'}
			}

			return b
		} else {
			//var buf []byte
			//buf = strconv.AppendInt(buf, numInt, 10)
			//return buf
			return []byte(strconv.FormatInt(numInt, 10))
		}
		// TODO - Overflow
	}
	return b
}

// Unoptimized since it's a rare feature
func writeHex(b []byte) []byte {

	// need to slice off initial "0x" or "0X"
	s := string(b[2:])
	numInt, err := strconv.ParseInt(s, 16, 64)
	if err == nil {
		buf := []byte{}
		buf = strconv.AppendInt(buf, numInt, 10)
		return buf
		//return []byte(strconv.FormatInt(numInt, 10))
	}
	// TODO - Overflow
	return b
}

func isFloat(b []byte) bool {

	if b[0] == '+' {
		return false
	}
	if b[0] == '-' {
		b = b[1:]
	}

	// non-standard
	if b[0] == '.' {
		return false
	}

	if b[0] == '0' && len(b) > 1 && b[1] != '.' {
		return false
	}

	// [-]#
	return true

}
func writeFloat(b []byte) []byte {
	if isFloat(b) {
		return b
	}

	s := string(b)

	// TODO
	// https://cs.opensource.google/go/go/+/refs/tags/go1.24.3:src/encoding/json/encode.go
	num, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return []byte(fmt.Sprintf("%g", num))
	}

	// TODO - overflow
	return b
}

func bareword(b []byte) []byte {
	if isNull(b) || isTrue(b) || isFalse(b) {
		return b
	}

	// TODO Nan
	// TODO Infinity

	// something else
	return b
}

// writeString takes an "quoted string with escapes" and converts to a JSON-spec string.
// it needs to handle
//
//   * single quote strings
//   * double quote strings
//   * backtick quote strings
//
// all with a variety of escape sequences.
//
//
func writeString(out *bytes.Buffer, src []byte) {
	// get quote type
	qchar := src[0]

	// strip off quotes
	src := src[1:len(src)-1]

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

	skip := false
	for _, b := range 

	if qchar == backQuote {
		// no need to unescape first
		// directly encode
		writeQuote(string(b[1:len(b)-1]), out)
		return
	}

	// TERRIBLE
	//
	// Unquote can fail if
	//  - missing starting or ending quotes
	//  - contains embedded raw newline
	bs := string(b)
	bs = strings.ReplaceAll(bs, "\n", "\\n")
	val, err := strconv.Unquote(bs)
	if err != nil {
		log.Fatalf("strconv.Unquote failed unexpectedly: %v", err)
	}
	writeQuote(val,out)
	return
}

var zeroCharArray = []byte{'0'}

var objectStart = []byte{'{'}
var objectEnd = []byte{'}'}

var arrayStart = []byte{'['}
var arrayEnd = []byte{']'}

func writeQuoted(src []byte, out *bytes.Buffer) {
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
