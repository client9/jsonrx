package tojson

import (
	"bytes"
	"fmt"
)

// --------------------------------------------------------------------------
// YAML parser options
// --------------------------------------------------------------------------

// yamlTabWidth is the number of spaces a tab character counts as when
// measuring indentation. Set to < 0 to forbid tabs in YAML input entirely.
const yamlTabWidth = 2

// yamlBoolAliases controls whether YAML 1.1 boolean aliases are recognised.
// When true, yes/no/on/off (and their case variants) map to true/false.
// When false, only true/false (and their case variants) are treated as booleans.
const yamlBoolAliases = true

// yamlTildeNull controls whether bare ~ is treated as null.
const yamlTildeNull = false

// writeScalar converts a YAML scalar to its JSON representation.
func writeScalar(s []byte, buf *bytes.Buffer) error {
	s = bytes.TrimSpace(s)
	switch string(s) {
	case "", "null", "Null", "NULL":
		buf.WriteString("null")
		return nil
	}
	if yamlTildeNull && string(s) == "~" {
		buf.WriteString("null")
		return nil
	}
	switch string(s) {
	case "true", "True", "TRUE":
		buf.WriteString("true")
		return nil
	case "false", "False", "FALSE":
		buf.WriteString("false")
		return nil
	}
	if yamlBoolAliases {
		switch string(s) {
		case "yes", "Yes", "YES", "on", "On", "ON":
			buf.WriteString("true")
			return nil
		case "no", "No", "NO", "off", "Off", "OFF":
			buf.WriteString("false")
			return nil
		}
	}

	if len(s) > 0 && s[0] == '"' {
		str, _, err := parseDoubleQuoted(s)
		if err != nil {
			return err
		}
		writeJSONString(str, buf)
		return nil
	}
	if len(s) > 0 && s[0] == '\'' {
		str := parseSingleQuoted(s)
		writeJSONString(str, buf)
		return nil
	}

	if isYAMLNumber(s) {
		buf.Write(s)
		return nil
	}

	writeJSONString(s, buf)
	return nil
}

// writeJSONString writes s as a properly escaped JSON string.
// Uses AvailableBuffer so that when buf has spare capacity no allocation is needed.
func writeJSONString(s []byte, buf *bytes.Buffer) {
	buf.Write(appendString(buf.AvailableBuffer(), s))
}

// --------------------------------------------------------------------------
// Quoted string parsers
// --------------------------------------------------------------------------

func parseUnicodeEscape(hex4 []byte) (rune, error) {
	var r rune
	for _, c := range hex4 {
		r <<= 4
		switch {
		case c >= '0' && c <= '9':
			r |= rune(c - '0')
		case c >= 'a' && c <= 'f':
			r |= rune(c-'a') + 10
		case c >= 'A' && c <= 'F':
			r |= rune(c-'A') + 10
		default:
			return 0, fmt.Errorf("invalid hex digit %q", c)
		}
	}
	return r, nil
}

// parseDoubleQuoted parses a double-quoted YAML string starting at s[0].
// Returns the unescaped content, the index after the closing '"', and any error.
func parseDoubleQuoted(s []byte) ([]byte, int, error) {
	if len(s) < 2 || s[0] != '"' {
		return s, len(s), nil
	}
	// Fast path: no escape sequences — return a no-alloc sub-slice.
	for i := 1; i < len(s); i++ {
		if s[i] == '"' {
			return s[1:i], i + 1, nil
		}
		if s[i] == '\\' {
			break // has escapes, fall through to slow path
		}
	}
	// Slow path: has escape sequences, must decode.
	var b bytes.Buffer
	i := 1
	for i < len(s) {
		c := s[i]
		if c == '"' {
			return b.Bytes(), i + 1, nil
		}
		if c == '\\' && i+1 < len(s) {
			i++
			switch s[i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			case '/':
				b.WriteByte('/')
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'u':
				if i+4 < len(s) {
					r, err := parseUnicodeEscape(s[i+1 : i+5])
					if err == nil {
						if r >= 0xD800 && r <= 0xDBFF && i+10 < len(s) && s[i+5] == '\\' && s[i+6] == 'u' {
							r2, err2 := parseUnicodeEscape(s[i+7 : i+11])
							if err2 == nil && r2 >= 0xDC00 && r2 <= 0xDFFF {
								r = 0x10000 + (r-0xD800)<<10 + (r2 - 0xDC00)
								i += 6
							}
						}
						b.WriteRune(r)
						i += 4
						break
					}
				}
				b.WriteString(`\u`)
			default:
				b.WriteByte('\\')
				b.WriteByte(s[i])
			}
		} else {
			b.WriteByte(c)
		}
		i++
	}
	return b.Bytes(), i, fmt.Errorf("unterminated double-quoted string")
}

func parseSingleQuoted(s []byte) []byte {
	str, _ := parseSingleQuotedRaw(s)
	return str
}

// --------------------------------------------------------------------------
// Line classification helpers
// --------------------------------------------------------------------------

func isSeqItem(content []byte) bool {
	return bytes.Equal(content, []byte("-")) || bytes.HasPrefix(content, []byte("- "))
}

// isMapKey returns true if content looks like a YAML mapping key line.
func isMapKey(content []byte) bool {
	if isSeqItem(content) {
		return false
	}
	if len(content) == 0 {
		return false
	}
	if content[0] == '{' || content[0] == '[' {
		return false
	}
	switch content[0] {
	case '"':
		i := 1
		for i < len(content) {
			if content[i] == '"' {
				return i+1 < len(content) && content[i+1] == ':'
			}
			if content[i] == '\\' {
				i++
			}
			i++
		}
		return false
	case '\'':
		i := 1
		for i < len(content) {
			if content[i] == '\'' {
				if i+1 < len(content) && content[i+1] == '\'' {
					i += 2
					continue
				}
				return i+1 < len(content) && content[i+1] == ':'
			}
			i++
		}
		return false
	}
	return bytes.Contains(content, []byte(": ")) || (len(content) > 0 && content[len(content)-1] == ':')
}

// splitMapKey splits "key: value" → ("key", "value"), or "key:" → ("key", nil).
func splitMapKey(content []byte) (key, value []byte) {
	switch {
	case len(content) > 0 && content[0] == '"':
		k, rest := parseDoubleQuotedRaw(content)
		rest = bytes.TrimPrefix(rest, []byte(":"))
		rest = bytes.TrimPrefix(rest, []byte(" "))
		return k, bytes.TrimSpace(rest)
	case len(content) > 0 && content[0] == '\'':
		k, rest := parseSingleQuotedRaw(content)
		rest = bytes.TrimPrefix(rest, []byte(":"))
		rest = bytes.TrimPrefix(rest, []byte(" "))
		return k, bytes.TrimSpace(rest)
	}
	if idx := bytes.Index(content, []byte(": ")); idx >= 0 {
		return content[:idx], bytes.TrimSpace(content[idx+2:])
	}
	if len(content) > 0 && content[len(content)-1] == ':' {
		return content[:len(content)-1], nil
	}
	return content, nil
}

// parseDoubleQuotedRaw returns (unescaped bytes, remainder after closing quote).
func parseDoubleQuotedRaw(s []byte) ([]byte, []byte) {
	str, end, _ := parseDoubleQuoted(s)
	return str, s[end:]
}

// parseSingleQuotedRaw returns (unescaped bytes, remainder after closing quote).
func parseSingleQuotedRaw(s []byte) ([]byte, []byte) {
	if len(s) < 2 || s[0] != '\'' {
		return s, nil
	}
	// Fast path: no '' escape sequences — return a no-alloc sub-slice.
	for i := 1; i < len(s); i++ {
		if s[i] == '\'' {
			if i+1 < len(s) && s[i+1] == '\'' {
				break // has '' escape, fall through to slow path
			}
			return s[1:i], s[i+1:]
		}
	}
	// Slow path: has '' escapes, must decode.
	var b bytes.Buffer
	i := 1
	for i < len(s) {
		if s[i] == '\'' {
			if i+1 < len(s) && s[i+1] == '\'' {
				b.WriteByte('\'')
				i += 2
				continue
			}
			return b.Bytes(), s[i+1:]
		}
		b.WriteByte(s[i])
		i++
	}
	return b.Bytes(), nil
}

// --------------------------------------------------------------------------
// Misc helpers
// --------------------------------------------------------------------------

func leadingSpaces(s []byte) int {
	n := 0
	for _, c := range s {
		if c == ' ' {
			n++
		} else if c == '\t' {
			n += 2
		} else {
			break
		}
	}
	return n
}

// yamlLeadingIndent counts the indentation of s using yamlTabWidth for tabs.
// Returns an error if yamlTabWidth < 0 and s contains a leading tab.
func yamlLeadingIndent(s []byte) (int, error) {
	n := 0
	for _, c := range s {
		if c == ' ' {
			n++
		} else if c == '\t' {
			if yamlTabWidth < 0 {
				return 0, fmt.Errorf("tab character not allowed in YAML indentation")
			}
			n += yamlTabWidth
		} else {
			break
		}
	}
	return n, nil
}

// isYAMLNumber returns true for byte slices that are valid JSON numbers (with optional + prefix).
func isYAMLNumber(s []byte) bool {
	if len(s) == 0 {
		return false
	}
	i := 0
	if s[i] == '-' || s[i] == '+' {
		i++
	}
	if i >= len(s) {
		return false
	}
	hasDigit := false
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		hasDigit = true
		i++
	}
	if !hasDigit {
		return false
	}
	if i < len(s) && s[i] == '.' {
		i++
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
	}
	return i == len(s)
}

// stripInlineComment removes a # comment from a content line, respecting quotes.
func stripInlineComment(s []byte) []byte {
	inDouble := false
	inSingle := false
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
			if c == '\'' && i+1 < len(s) && s[i+1] == '\'' {
				i++
			} else if c == '\'' {
				inSingle = false
			}
		case c == '"':
			inDouble = true
		case c == '\'':
			inSingle = true
		case c == '#':
			if i > 0 && (s[i-1] == ' ' || s[i-1] == '\t') {
				return bytes.TrimRight(s[:i], " \t")
			}
		}
	}
	return s
}
