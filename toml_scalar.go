package tojson

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// scalarStringNode encodes s as a JSON string and returns it as a scalar jnode.
func scalarStringNode(s string) *jnode {
	return &jnode{raw: appendStringStr(make([]byte, 0, len(s)+2), s)}
}

// parseTOMLValue parses a TOML value from s.
// rawLines/lineIdx are needed for multiline strings.
// Returns pre-encoded JSON bytes, number of additional lines consumed, and error.
func parseTOMLValue(s string, rawLines []string, lineIdx int) (*jnode, int, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return nil, 0, fmt.Errorf("expected value")
	}

	// multiline basic string
	if strings.HasPrefix(s, `"""`) {
		str, consumed, err := parseTOMLMultilineBasic(s, rawLines, lineIdx)
		if err != nil {
			return nil, 0, err
		}
		return scalarStringNode(str), consumed, nil
	}

	// basic string
	if s[0] == '"' {
		str, _, err := parseTOMLBasicStringRaw(s)
		if err != nil {
			return nil, 0, err
		}
		return scalarStringNode(str), 0, nil
	}

	// multiline literal string
	if strings.HasPrefix(s, "'''") {
		str, consumed, err := parseTOMLMultilineLiteral(s, rawLines, lineIdx)
		if err != nil {
			return nil, 0, err
		}
		return scalarStringNode(str), consumed, nil
	}

	// literal string
	if s[0] == '\'' {
		str, _ := parseTOMLLiteralStringRaw(s)
		return scalarStringNode(str), 0, nil
	}

	// inline table
	if s[0] == '{' {
		node, _, err := parseTOMLInlineTable(s, 0)
		if err != nil {
			return nil, 0, err
		}
		return node, 0, nil
	}

	// inline array
	if s[0] == '[' {
		node, _, consumed, err := parseTOMLInlineArray(s, rawLines, lineIdx, 0)
		if err != nil {
			return nil, 0, err
		}
		return node, consumed, nil
	}

	// booleans (TOML: lowercase only)
	if s == "true" {
		return nodeTrue, 0, nil
	}
	if s == "false" {
		return nodeFalse, 0, nil
	}

	// inf / nan
	if strings.EqualFold(s, "inf") || strings.EqualFold(s, "+inf") || strings.EqualFold(s, "-inf") {
		return nil, 0, fmt.Errorf("inf is not representable in JSON")
	}
	if strings.EqualFold(s, "nan") || strings.EqualFold(s, "+nan") || strings.EqualFold(s, "-nan") {
		return nil, 0, fmt.Errorf("nan is not representable in JSON")
	}

	// datetime detection: YYYY-... or HH:MM
	if isTOMLDateTime(s) {
		return scalarStringNode(s), 0, nil
	}

	// number
	raw, err := parseTOMLNumber(s)
	if err != nil {
		return nil, 0, err
	}
	return newScalarNode(raw), 0, nil
}

// --------------------------------------------------------------------------
// String parsers
// --------------------------------------------------------------------------

// applyTOMLEscape processes a TOML escape sequence. i points to the character
// immediately after the backslash within s. The decoded rune is written to b.
// Returns the number of additional characters consumed beyond s[i], or an error.
func applyTOMLEscape(s string, i int, b *strings.Builder) (int, error) {
	if i >= len(s) {
		return 0, fmt.Errorf("unexpected end of string after backslash")
	}
	switch s[i] {
	case 'b':
		b.WriteByte('\b')
	case 't':
		b.WriteByte('\t')
	case 'n':
		b.WriteByte('\n')
	case 'f':
		b.WriteByte('\f')
	case 'r':
		b.WriteByte('\r')
	case '"':
		b.WriteByte('"')
	case '\\':
		b.WriteByte('\\')
	case 'u': // \uXXXX
		if i+4 < len(s) {
			r, err := parseUnicodeEscape(s[i+1 : i+5])
			if err == nil {
				// surrogate pair \uHigh\uLow
				if r >= 0xD800 && r <= 0xDBFF && i+10 < len(s) && s[i+5] == '\\' && s[i+6] == 'u' {
					r2, err2 := parseUnicodeEscape(s[i+7 : i+11])
					if err2 == nil && r2 >= 0xDC00 && r2 <= 0xDFFF {
						r = 0x10000 + (r-0xD800)<<10 + (r2 - 0xDC00)
						b.WriteRune(r)
						return 10, nil
					}
				}
				b.WriteRune(r)
				return 4, nil
			}
		}
		return 0, fmt.Errorf("invalid \\u escape")
	case 'U': // \UXXXXXXXX
		if i+8 < len(s) {
			hi, err1 := parseUnicodeEscape(s[i+1 : i+5])
			lo, err2 := parseUnicodeEscape(s[i+5 : i+9])
			if err1 == nil && err2 == nil {
				r := (rune(hi) << 16) | rune(lo)
				if utf8.ValidRune(r) {
					b.WriteRune(r)
					return 8, nil
				}
			}
		}
		return 0, fmt.Errorf("invalid \\U escape")
	default:
		return 0, fmt.Errorf("invalid escape \\%c", s[i])
	}
	return 0, nil
}

// parseTOMLBasicStringRaw parses a TOML basic (double-quoted) string from s[0].
// Returns the decoded string and the remainder after the closing quote.
func parseTOMLBasicStringRaw(s string) (string, string, error) {
	if len(s) < 2 || s[0] != '"' {
		return "", s, fmt.Errorf("expected double-quoted string")
	}
	// Fast path: no escape sequences — return a no-alloc substring.
	for i := 1; i < len(s); i++ {
		if s[i] == '"' {
			return s[1:i], s[i+1:], nil
		}
		if s[i] == '\\' {
			break
		}
	}
	// Slow path: has escape sequences, must decode.
	var b strings.Builder
	i := 1
	for i < len(s) {
		c := s[i]
		if c == '"' {
			return b.String(), s[i+1:], nil
		}
		if c == '\\' && i+1 < len(s) {
			i++
			extra, err := applyTOMLEscape(s, i, &b)
			if err != nil {
				return "", s, err
			}
			i += extra
		} else {
			b.WriteByte(c)
		}
		i++
	}
	return "", s, fmt.Errorf("unterminated basic string")
}

// parseTOMLLiteralStringRaw parses a TOML literal (single-quoted) string from s[0].
// No escape processing. Returns the raw content and the remainder.
func parseTOMLLiteralStringRaw(s string) (string, string) {
	if len(s) < 2 || s[0] != '\'' {
		return s, ""
	}
	i := 1
	for i < len(s) {
		if s[i] == '\'' {
			return s[1:i], s[i+1:]
		}
		i++
	}
	// unterminated — return what we have
	return s[1:], ""
}

// parseTOMLMultilineBasic parses a triple-double-quoted multiline basic string.
// s is the portion of the current line starting at the opening """.
// Returns the decoded string and the number of additional lines consumed.
func parseTOMLMultilineBasic(s string, rawLines []string, lineIdx int) (string, int, error) {
	if !strings.HasPrefix(s, `"""`) {
		return "", 0, fmt.Errorf("expected \"\"\"")
	}
	// combine lines into one string for scanning
	content := s[3:] // skip opening """
	extraLines := 0
	for {
		if idx := strings.Index(content, `"""`); idx >= 0 {
			body := content[:idx]
			str, _, err := decodeTOMLMultilineBasic(body)
			return str, extraLines, err
		}
		// need more lines
		nextIdx := lineIdx + extraLines + 1
		if nextIdx >= len(rawLines) {
			return "", 0, fmt.Errorf("unterminated multiline basic string")
		}
		content += "\n" + rawLines[nextIdx]
		extraLines++
	}
}

func decodeTOMLMultilineBasic(s string) (string, int, error) {
	// trim a single leading newline per spec
	if strings.HasPrefix(s, "\n") {
		s = s[1:]
	} else if strings.HasPrefix(s, "\r\n") {
		s = s[2:]
	}
	var b strings.Builder
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			next := s[i+1]
			// line-ending backslash: trim following whitespace/newlines
			if next == '\n' || next == '\r' || next == ' ' || next == '\t' {
				i++
				for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
					i++
				}
				continue
			}
			i++ // advance past backslash to escape char
			extra, err := applyTOMLEscape(s, i, &b)
			if err != nil {
				return "", 0, err
			}
			i += extra
		} else {
			b.WriteByte(c)
		}
		i++
	}
	return b.String(), 0, nil
}

// parseTOMLMultilineLiteral parses a triple-single-quoted multiline literal string.
func parseTOMLMultilineLiteral(s string, rawLines []string, lineIdx int) (string, int, error) {
	if !strings.HasPrefix(s, "'''") {
		return "", 0, fmt.Errorf("expected '''")
	}
	content := s[3:]
	extraLines := 0
	for {
		if idx := strings.Index(content, "'''"); idx >= 0 {
			body := content[:idx]
			// trim single leading newline per spec
			if strings.HasPrefix(body, "\n") {
				body = body[1:]
			} else if strings.HasPrefix(body, "\r\n") {
				body = body[2:]
			}
			return body, extraLines, nil
		}
		nextIdx := lineIdx + extraLines + 1
		if nextIdx >= len(rawLines) {
			return "", 0, fmt.Errorf("unterminated multiline literal string")
		}
		content += "\n" + rawLines[nextIdx]
		extraLines++
	}
}

// --------------------------------------------------------------------------
// Number parser
// --------------------------------------------------------------------------

func parseTOMLNumber(s string) ([]byte, error) {
	orig := s

	// extract sign
	sign := ""
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		sign = string(s[0])
		s = s[1:]
	}
	if len(s) == 0 {
		return nil, fmt.Errorf("invalid number: %s", orig)
	}

	// radix prefixes (no sign allowed before 0x/0o/0b per TOML spec)
	if sign == "" {
		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			digits := stripUnderscores(s[2:], orig)
			if digits == "" {
				return nil, fmt.Errorf("invalid hex number: %s", orig)
			}
			v, err := strconv.ParseInt(digits, 16, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid hex number %s: %v", orig, err)
			}
			return []byte(strconv.FormatInt(v, 10)), nil
		}
		if strings.HasPrefix(s, "0o") || strings.HasPrefix(s, "0O") {
			digits := stripUnderscores(s[2:], orig)
			v, err := strconv.ParseInt(digits, 8, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid octal number %s: %v", orig, err)
			}
			return []byte(strconv.FormatInt(v, 10)), nil
		}
		if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
			digits := stripUnderscores(s[2:], orig)
			v, err := strconv.ParseInt(digits, 2, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid binary number %s: %v", orig, err)
			}
			return []byte(strconv.FormatInt(v, 10)), nil
		}
	}

	// decimal integer or float — strip underscores
	stripped, err := stripUnderscoresValidated(s)
	if err != nil {
		return nil, fmt.Errorf("invalid number %s: %v", orig, err)
	}
	s = stripped

	// check for leading zeros (not allowed except bare "0")
	if len(s) > 1 && s[0] == '0' && s[1] >= '0' && s[1] <= '9' {
		return nil, fmt.Errorf("leading zeros not allowed in integer: %s", orig)
	}

	isFloat := strings.ContainsAny(s, ".eE")
	// JSON does not allow + prefix; strip it
	if sign == "+" {
		sign = ""
	}
	result := sign + s

	if isFloat {
		// validate by parsing
		if _, err := strconv.ParseFloat(result, 64); err != nil {
			return nil, fmt.Errorf("invalid float %s: %v", orig, err)
		}
		return []byte(result), nil
	}

	// integer: validate
	if _, err := strconv.ParseInt(result, 10, 64); err != nil {
		// try uint64 for large positive integers
		if _, err2 := strconv.ParseUint(result, 10, 64); err2 != nil {
			return nil, fmt.Errorf("invalid integer %s: %v", orig, err)
		}
	}
	return []byte(result), nil
}

// stripUnderscores removes underscores without validation (for radix-prefixed numbers).
func stripUnderscores(s, orig string) string {
	return strings.ReplaceAll(s, "_", "")
}

// stripUnderscoresValidated removes underscores from a decimal/float string,
// validating that they are not at the start, end, or adjacent.
func stripUnderscoresValidated(s string) (string, error) {
	if !strings.Contains(s, "_") {
		return s, nil
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '_' {
			if i == 0 || i == len(s)-1 {
				return "", fmt.Errorf("underscore at start or end of number")
			}
			if s[i-1] == '_' {
				return "", fmt.Errorf("adjacent underscores in number")
			}
			if s[i-1] == '.' || (i+1 < len(s) && s[i+1] == '.') {
				return "", fmt.Errorf("underscore adjacent to decimal point")
			}
			if s[i-1] == 'e' || s[i-1] == 'E' || (i+1 < len(s) && (s[i+1] == 'e' || s[i+1] == 'E')) {
				return "", fmt.Errorf("underscore adjacent to exponent")
			}
			continue
		}
		b.WriteByte(c)
	}
	return b.String(), nil
}

// --------------------------------------------------------------------------
// Datetime detection
// --------------------------------------------------------------------------

func isTOMLDateTime(s string) bool {
	// local time: HH:MM:...
	if len(s) >= 5 && s[2] == ':' && isDigits(s[0:2]) {
		return true
	}
	// date / datetime: YYYY-...
	if len(s) >= 10 && s[4] == '-' && isDigits(s[0:4]) {
		return true
	}
	return false
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// --------------------------------------------------------------------------
// Inline table parser
// --------------------------------------------------------------------------

// parseTOMLInlineTable parses {k = v, ...} starting at s[pos].
// Returns the built jnode, position after the closing '}', and any error.
func parseTOMLInlineTable(s string, pos int) (*jnode, int, error) {
	if pos >= len(s) || s[pos] != '{' {
		return nil, pos, fmt.Errorf("expected '{'")
	}
	pos++ // consume '{'
	node := newObjectNode()
	pos = flowSkipWS(s, pos)

	if pos < len(s) && s[pos] == '}' {
		return node, pos + 1, nil
	}

	first := true
	for pos < len(s) {
		if !first {
			if pos >= len(s) || s[pos] != ',' {
				return nil, pos, fmt.Errorf("expected ',' or '}' in inline table")
			}
			pos = flowSkipWS(s, pos+1)
			// no trailing comma in TOML inline tables
			if pos < len(s) && s[pos] == '}' {
				return nil, pos, fmt.Errorf("trailing comma not allowed in inline table")
			}
		}
		first = false

		path, rest, err := parseTOMLKeyPath(s[pos:])
		if err != nil {
			return nil, pos, err
		}
		pos += len(s[pos:]) - len(rest)
		pos = flowSkipWS(s, pos)
		if pos >= len(s) || s[pos] != '=' {
			return nil, pos, fmt.Errorf("expected '=' in inline table")
		}
		pos = flowSkipWS(s, pos+1)

		valNode, newPos, err := parseTOMLInlineValue(s, pos)
		if err != nil {
			return nil, pos, err
		}
		pos = flowSkipWS(s, newPos)

		// navigate dotted path
		target := node
		for i := 0; i < len(path)-1; i++ {
			pair := target.findPair(path[i])
			if pair == nil {
				next := newObjectNode()
				target.obj = append(target.obj, &jpair{key: path[i], val: next})
				target = next
			} else if pair.val.obj != nil {
				target = pair.val
			} else {
				return nil, pos, fmt.Errorf("duplicate key %q in inline table", path[i])
			}
		}
		lastKey := path[len(path)-1]
		if target.findPair(lastKey) != nil {
			return nil, pos, fmt.Errorf("duplicate key %q in inline table", lastKey)
		}
		target.obj = append(target.obj, &jpair{key: lastKey, val: valNode})

		if pos < len(s) && s[pos] == '}' {
			return node, pos + 1, nil
		}
	}
	return nil, pos, fmt.Errorf("unterminated inline table")
}

// parseTOMLInlineValue parses a single value inside an inline collection (no multiline).
func parseTOMLInlineValue(s string, pos int) (*jnode, int, error) {
	pos = flowSkipWS(s, pos)
	if pos >= len(s) {
		return nil, pos, fmt.Errorf("expected value")
	}
	rest := s[pos:]
	end := tomlValueEnd(rest)
	node, _, err := parseTOMLValue(rest[:end], nil, 0)
	if err != nil {
		return nil, pos, err
	}
	return node, pos + end, nil
}

// tomlValueEnd returns the number of bytes in s consumed by the first TOML value.
// Used after parseTOMLValue to advance the position cursor.
func tomlValueEnd(s string) int {
	s = strings.TrimLeft(s, " \t")
	if len(s) == 0 {
		return 0
	}
	switch {
	case strings.HasPrefix(s, `"""`):
		i := 3
		for i < len(s) {
			if strings.HasPrefix(s[i:], `"""`) {
				return i + 3
			}
			if s[i] == '\\' {
				i += 2
			} else {
				i++
			}
		}
		return len(s)
	case s[0] == '"':
		i := 1
		for i < len(s) {
			if s[i] == '\\' {
				i += 2
			} else if s[i] == '"' {
				return i + 1
			} else {
				i++
			}
		}
		return len(s)
	case strings.HasPrefix(s, "'''"):
		i := 3
		for i < len(s) {
			if strings.HasPrefix(s[i:], "'''") {
				return i + 3
			}
			i++
		}
		return len(s)
	case s[0] == '\'':
		i := 1
		for i < len(s) {
			if s[i] == '\'' {
				return i + 1
			}
			i++
		}
		return len(s)
	case s[0] == '{':
		depth := 0
		inDouble, inSingle := false, false
		for i := 0; i < len(s); i++ {
			c := s[i]
			switch {
			case inDouble:
				if c == '\\' {
					i++
				} else if c == '"' {
					inDouble = false
				}
			case inSingle:
				if c == '\'' {
					inSingle = false
				}
			case c == '"':
				inDouble = true
			case c == '\'':
				inSingle = true
			case c == '{':
				depth++
			case c == '}':
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
		return len(s)
	case s[0] == '[':
		depth := 0
		inDouble, inSingle := false, false
		for i := 0; i < len(s); i++ {
			c := s[i]
			switch {
			case inDouble:
				if c == '\\' {
					i++
				} else if c == '"' {
					inDouble = false
				}
			case inSingle:
				if c == '\'' {
					inSingle = false
				}
			case c == '"':
				inDouble = true
			case c == '\'':
				inSingle = true
			case c == '[':
				depth++
			case c == ']':
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
		return len(s)
	default:
		// bare value: ends at , } ] whitespace or end
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c == ',' || c == '}' || c == ']' || c == ' ' || c == '\t' || c == '\r' || c == '\n' {
				return i
			}
		}
		return len(s)
	}
}

// --------------------------------------------------------------------------
// Direct-write value helpers (streaming path — no jnode allocation)
// --------------------------------------------------------------------------

// writeTOMLValue writes the JSON representation of a TOML value directly to buf
// without allocating an intermediate jnode.
// Returns extra lines consumed (for multiline strings) and any error.
func writeTOMLValue(s string, rawLines []string, lineIdx int, buf *bytes.Buffer) (int, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return 0, fmt.Errorf("expected value")
	}

	if strings.HasPrefix(s, `"""`) {
		str, consumed, err := parseTOMLMultilineBasic(s, rawLines, lineIdx)
		if err != nil {
			return 0, err
		}
		writeJSONString(str, buf)
		return consumed, nil
	}
	if s[0] == '"' {
		str, _, err := parseTOMLBasicStringRaw(s)
		if err != nil {
			return 0, err
		}
		writeJSONString(str, buf)
		return 0, nil
	}
	if strings.HasPrefix(s, "'''") {
		str, consumed, err := parseTOMLMultilineLiteral(s, rawLines, lineIdx)
		if err != nil {
			return 0, err
		}
		writeJSONString(str, buf)
		return consumed, nil
	}
	if s[0] == '\'' {
		str, _ := parseTOMLLiteralStringRaw(s)
		writeJSONString(str, buf)
		return 0, nil
	}
	if s[0] == '{' {
		node, _, err := parseTOMLInlineTable(s, 0)
		if err != nil {
			return 0, err
		}
		serializeNode(node, buf)
		return 0, nil
	}
	if s[0] == '[' {
		return writeTOMLInlineArray(s, rawLines, lineIdx, buf)
	}
	if s == "true" {
		buf.WriteString("true")
		return 0, nil
	}
	if s == "false" {
		buf.WriteString("false")
		return 0, nil
	}
	if strings.EqualFold(s, "inf") || strings.EqualFold(s, "+inf") || strings.EqualFold(s, "-inf") {
		return 0, fmt.Errorf("inf is not representable in JSON")
	}
	if strings.EqualFold(s, "nan") || strings.EqualFold(s, "+nan") || strings.EqualFold(s, "-nan") {
		return 0, fmt.Errorf("nan is not representable in JSON")
	}
	if isTOMLDateTime(s) {
		writeJSONString(s, buf)
		return 0, nil
	}
	// fast path: valid JSON number as-is (no + prefix, no leading zeros, no underscores/radix)
	if isYAMLNumber(s) && s[0] != '+' {
		digits := s
		if digits[0] == '-' {
			digits = digits[1:]
		}
		if len(digits) < 2 || digits[0] != '0' || digits[1] < '0' || digits[1] > '9' {
			buf.WriteString(s)
			return 0, nil
		}
	}
	// strip leading + and retry (parseTOMLNumber handles validation)
	if len(s) > 1 && s[0] == '+' && isYAMLNumber(s[1:]) {
		s2 := s[1:]
		digits := s2
		if len(digits) < 2 || digits[0] != '0' || digits[1] < '0' || digits[1] > '9' {
			buf.WriteString(s2)
			return 0, nil
		}
	}
	// full parse (handles underscores, radix prefixes, leading-zero errors)
	raw, err := parseTOMLNumber(s)
	if err != nil {
		return 0, err
	}
	buf.Write(raw)
	return 0, nil
}

// writeTOMLInlineArray writes [v, v, ...] starting at s[0] directly to buf,
// mirroring parseTOMLInlineArray without building a jnode tree.
func writeTOMLInlineArray(s string, rawLines []string, lineIdx int, buf *bytes.Buffer) (int, error) {
	pos := 1 // consume '['
	extraLines := 0
	buf.WriteByte('[')
	count := 0

	for {
		for {
			pos = flowSkipWS(s, pos)
			if pos < len(s) {
				break
			}
			nextIdx := lineIdx + extraLines + 1
			if rawLines == nil || nextIdx >= len(rawLines) {
				return extraLines, fmt.Errorf("unterminated inline array")
			}
			s += "\n" + rawLines[nextIdx]
			extraLines++
		}

		if s[pos] == ']' {
			buf.WriteByte(']')
			return extraLines, nil
		}

		if count > 0 {
			if s[pos] != ',' {
				return extraLines, fmt.Errorf("expected ',' or ']' in array")
			}
			pos = flowSkipWS(s, pos+1)
			for {
				pos = flowSkipWS(s, pos)
				if pos < len(s) {
					break
				}
				nextIdx := lineIdx + extraLines + 1
				if rawLines == nil || nextIdx >= len(rawLines) {
					return extraLines, fmt.Errorf("unterminated inline array")
				}
				s += "\n" + rawLines[nextIdx]
				extraLines++
			}
			if s[pos] == ']' {
				buf.WriteByte(']')
				return extraLines, nil
			}
			buf.WriteByte(',')
		}

		rest := strings.TrimLeft(s[pos:], " \t")
		lead := len(s[pos:]) - len(rest)
		valEnd := tomlValueEnd(rest)
		consumed, err := writeTOMLValue(rest[:valEnd], rawLines, lineIdx+extraLines, buf)
		if err != nil {
			return extraLines, err
		}
		extraLines += consumed
		pos = pos + lead + valEnd
		count++
	}
}

// --------------------------------------------------------------------------
// Inline array parser
// --------------------------------------------------------------------------

// parseTOMLInlineArray parses [v, v, ...] starting at s[pos].
// May span multiple lines. Returns the built jnode, position after ']',
// lines consumed beyond lineIdx, and any error.
func parseTOMLInlineArray(s string, rawLines []string, lineIdx int, pos int) (*jnode, int, int, error) {
	if pos >= len(s) || s[pos] != '[' {
		return nil, pos, 0, fmt.Errorf("expected '['")
	}
	pos++ // consume '['
	extraLines := 0
	node := &jnode{arr: []*jnode{}}

	for {
		// skip whitespace and newlines, pulling in more rawLines as needed
		for {
			pos = flowSkipWS(s, pos)
			if pos < len(s) {
				break
			}
			// pull next line
			nextIdx := lineIdx + extraLines + 1
			if rawLines == nil || nextIdx >= len(rawLines) {
				return nil, pos, extraLines, fmt.Errorf("unterminated inline array")
			}
			s += "\n" + rawLines[nextIdx]
			extraLines++
		}

		if s[pos] == ']' {
			return node, pos + 1, extraLines, nil
		}

		if len(node.arr) > 0 {
			if s[pos] != ',' {
				return nil, pos, extraLines, fmt.Errorf("expected ',' or ']' in array")
			}
			pos = flowSkipWS(s, pos+1)
			// skip whitespace/newlines after comma
			for {
				pos = flowSkipWS(s, pos)
				if pos < len(s) {
					break
				}
				nextIdx := lineIdx + extraLines + 1
				if rawLines == nil || nextIdx >= len(rawLines) {
					return nil, pos, extraLines, fmt.Errorf("unterminated inline array")
				}
				s += "\n" + rawLines[nextIdx]
				extraLines++
			}
			// trailing comma allowed
			if s[pos] == ']' {
				return node, pos + 1, extraLines, nil
			}
		}

		rest := strings.TrimLeft(s[pos:], " \t")
		lead := len(s[pos:]) - len(rest)
		valEnd := tomlValueEnd(rest)
		valNode, consumed, err := parseTOMLValue(rest[:valEnd], rawLines, lineIdx+extraLines)
		if err != nil {
			return nil, pos, extraLines, err
		}
		extraLines += consumed
		pos = pos + lead + valEnd
		node.arr = append(node.arr, valNode)
	}
}
