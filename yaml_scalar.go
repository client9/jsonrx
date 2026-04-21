package tojson

import (
	"bytes"
	"fmt"
	"strings"
)

// writeScalar converts a YAML scalar string to its JSON representation.
func writeScalar(s string, buf *bytes.Buffer) error {
	s = strings.TrimSpace(s)
	switch s {
	case "", "null", "~", "Null", "NULL":
		buf.WriteString("null")
		return nil
	case "true", "True", "TRUE", "yes", "Yes", "YES", "on", "On", "ON":
		buf.WriteString("true")
		return nil
	case "false", "False", "FALSE", "no", "No", "NO", "off", "Off", "OFF":
		buf.WriteString("false")
		return nil
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
		buf.WriteString(s)
		return nil
	}

	writeJSONString(s, buf)
	return nil
}

// writeJSONString writes s as a properly escaped JSON string.
// Uses AvailableBuffer so that when buf has spare capacity no allocation is needed.
func writeJSONString(s string, buf *bytes.Buffer) {
	buf.Write(appendStringStr(buf.AvailableBuffer(), s))
}

// --------------------------------------------------------------------------
// Quoted string parsers
// --------------------------------------------------------------------------

func parseUnicodeEscape(hex4 string) (rune, error) {
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
func parseDoubleQuoted(s string) (string, int, error) {
	if len(s) < 2 || s[0] != '"' {
		return s, len(s), nil
	}
	// Fast path: no escape sequences — return a no-alloc substring.
	for i := 1; i < len(s); i++ {
		if s[i] == '"' {
			return s[1:i], i + 1, nil
		}
		if s[i] == '\\' {
			break // has escapes, fall through to slow path
		}
	}
	// Slow path: has escape sequences, must decode.
	var b strings.Builder
	i := 1
	for i < len(s) {
		c := s[i]
		if c == '"' {
			return b.String(), i + 1, nil
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
	return b.String(), i, fmt.Errorf("unterminated double-quoted string")
}

func parseSingleQuoted(s string) string {
	str, _ := parseSingleQuotedRaw(s)
	return str
}

// --------------------------------------------------------------------------
// Line classification helpers
// --------------------------------------------------------------------------

func isSeqItem(content string) bool {
	return content == "-" || strings.HasPrefix(content, "- ")
}

// isMapKey returns true if content looks like a YAML mapping key line.
func isMapKey(content string) bool {
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
	return strings.Contains(content, ": ") || strings.HasSuffix(content, ":")
}

// splitMapKey splits "key: value" → ("key", "value"), or "key:" → ("key", "").
func splitMapKey(content string) (key, value string) {
	switch {
	case len(content) > 0 && content[0] == '"':
		k, rest := parseDoubleQuotedRaw(content)
		rest = strings.TrimPrefix(rest, ":")
		rest = strings.TrimPrefix(rest, " ")
		return k, strings.TrimSpace(rest)
	case len(content) > 0 && content[0] == '\'':
		k, rest := parseSingleQuotedRaw(content)
		rest = strings.TrimPrefix(rest, ":")
		rest = strings.TrimPrefix(rest, " ")
		return k, strings.TrimSpace(rest)
	}
	if idx := strings.Index(content, ": "); idx >= 0 {
		return content[:idx], strings.TrimSpace(content[idx+2:])
	}
	if strings.HasSuffix(content, ":") {
		return content[:len(content)-1], ""
	}
	return content, ""
}

// parseDoubleQuotedRaw returns (unescaped string, remainder after closing quote).
func parseDoubleQuotedRaw(s string) (string, string) {
	str, end, _ := parseDoubleQuoted(s)
	return str, s[end:]
}

// parseSingleQuotedRaw returns (string, remainder after closing quote).
func parseSingleQuotedRaw(s string) (string, string) {
	if len(s) < 2 || s[0] != '\'' {
		return s, ""
	}
	// Fast path: no '' escape sequences — return a no-alloc substring.
	for i := 1; i < len(s); i++ {
		if s[i] == '\'' {
			if i+1 < len(s) && s[i+1] == '\'' {
				break // has '' escape, fall through to slow path
			}
			return s[1:i], s[i+1:]
		}
	}
	// Slow path: has '' escapes, must decode.
	var b strings.Builder
	i := 1
	for i < len(s) {
		if s[i] == '\'' {
			if i+1 < len(s) && s[i+1] == '\'' {
				b.WriteByte('\'')
				i += 2
				continue
			}
			return b.String(), s[i+1:]
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String(), ""
}

// --------------------------------------------------------------------------
// Misc helpers
// --------------------------------------------------------------------------

func leadingSpaces(s string) int {
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

// isYAMLNumber returns true for strings that are valid JSON numbers.
func isYAMLNumber(s string) bool {
	if s == "" {
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
func stripInlineComment(s string) string {
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
				return strings.TrimRight(s[:i], " \t")
			}
		}
	}
	return s
}
