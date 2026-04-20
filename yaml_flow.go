package tojson

import (
	"bytes"
	"fmt"
	"strings"
)

// isFlowValue reports whether s begins with a flow mapping or sequence.
func isFlowValue(s string) bool {
	s = strings.TrimSpace(s)
	return len(s) > 0 && (s[0] == '{' || s[0] == '[')
}

// flowDepth returns the net count of open flow delimiters minus closed ones,
// ignoring content inside quoted strings.
func flowDepth(s string) int {
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
			if c == '\'' && i+1 < len(s) && s[i+1] == '\'' {
				i++
			} else if c == '\'' {
				inSingle = false
			}
		case c == '"':
			inDouble = true
		case c == '\'':
			inSingle = true
		case c == '{' || c == '[':
			depth++
		case c == '}' || c == ']':
			depth--
		}
	}
	return depth
}

// gatherFlowSrc builds a complete flow expression string starting with first.
// If brackets are unbalanced it reads additional rawLines to support multi-line
// flow values. Returns the assembled string and the last rawLine index consumed.
func (p *parser) gatherFlowSrc(first string, rawLineIdx int) (string, int) {
	var sb strings.Builder
	sb.WriteString(first)
	depth := flowDepth(first)
	last := rawLineIdx
	for depth > 0 {
		rawLineIdx++
		if rawLineIdx >= len(p.rawLines) {
			break
		}
		line := strings.TrimRight(p.rawLines[rawLineIdx], " \t\r")
		line = stripInlineComment(strings.TrimSpace(line))
		if line == "" {
			continue
		}
		sb.WriteByte(' ')
		sb.WriteString(line)
		depth += flowDepth(line)
		last = rawLineIdx
	}
	return sb.String(), last
}

// parseFlowExpr parses a complete YAML flow expression (mapping, sequence, or
// scalar) from s and writes its JSON representation to buf.
func parseFlowExpr(s string, buf *bytes.Buffer) error {
	s = strings.TrimSpace(s)
	switch {
	case len(s) == 0:
		buf.WriteString("null")
		return nil
	case s[0] == '{':
		_, err := parseFlowMapping(s, 0, buf)
		return err
	case s[0] == '[':
		_, err := parseFlowSequence(s, 0, buf)
		return err
	default:
		return writeScalar(s, buf)
	}
}

// parseFlowMapping parses a flow mapping starting at s[pos] (which must be '{').
func parseFlowMapping(s string, pos int, buf *bytes.Buffer) (int, error) {
	pos++ // consume '{'
	buf.WriteByte('{')
	pos = flowSkipWS(s, pos)
	first := true
	for pos < len(s) {
		if s[pos] == '}' {
			buf.WriteByte('}')
			return pos + 1, nil
		}
		if !first {
			if s[pos] != ',' {
				return pos, fmt.Errorf("expected ',' or '}' in flow mapping")
			}
			pos = flowSkipWS(s, pos+1)
			if pos < len(s) && s[pos] == '}' {
				buf.WriteByte('}')
				return pos + 1, nil
			}
			buf.WriteByte(',')
		}
		first = false

		key, newPos, err := flowParseKey(s, pos)
		if err != nil {
			return newPos, err
		}
		writeJSONString(key, buf)
		pos = flowSkipWS(s, newPos)
		if pos < len(s) && s[pos] == ':' {
			pos = flowSkipWS(s, pos+1)
		}
		buf.WriteByte(':')

		pos, err = flowParseItem(s, pos, buf)
		if err != nil {
			return pos, err
		}
		pos = flowSkipWS(s, pos)
	}
	return pos, fmt.Errorf("unterminated flow mapping")
}

// parseFlowSequence parses a flow sequence starting at s[pos] (which must be '[').
func parseFlowSequence(s string, pos int, buf *bytes.Buffer) (int, error) {
	pos++ // consume '['
	buf.WriteByte('[')
	pos = flowSkipWS(s, pos)
	first := true
	for pos < len(s) {
		if s[pos] == ']' {
			buf.WriteByte(']')
			return pos + 1, nil
		}
		if !first {
			if s[pos] != ',' {
				return pos, fmt.Errorf("expected ',' or ']' in flow sequence")
			}
			pos = flowSkipWS(s, pos+1)
			if pos < len(s) && s[pos] == ']' {
				buf.WriteByte(']')
				return pos + 1, nil
			}
			buf.WriteByte(',')
		}
		first = false

		var err error
		pos, err = flowParseItem(s, pos, buf)
		if err != nil {
			return pos, err
		}
		pos = flowSkipWS(s, pos)
	}
	return pos, fmt.Errorf("unterminated flow sequence")
}

// flowParseItem parses a single flow value (mapping, sequence, or scalar).
func flowParseItem(s string, pos int, buf *bytes.Buffer) (int, error) {
	pos = flowSkipWS(s, pos)
	if pos >= len(s) {
		buf.WriteString("null")
		return pos, nil
	}
	switch s[pos] {
	case '{':
		return parseFlowMapping(s, pos, buf)
	case '[':
		return parseFlowSequence(s, pos, buf)
	case '"':
		str, newPos, err := flowParseDoubleQuoted(s, pos)
		if err != nil {
			return newPos, err
		}
		writeJSONString(str, buf)
		return newPos, nil
	case '\'':
		str, newPos := flowParseSingleQuoted(s, pos)
		writeJSONString(str, buf)
		return newPos, nil
	default:
		start := pos
		for pos < len(s) && s[pos] != ',' && s[pos] != '}' && s[pos] != ']' {
			pos++
		}
		return pos, writeScalar(strings.TrimSpace(s[start:pos]), buf)
	}
}

// flowParseKey reads a mapping key (bare, double-quoted, or single-quoted).
func flowParseKey(s string, pos int) (string, int, error) {
	switch {
	case pos < len(s) && s[pos] == '"':
		str, newPos, err := flowParseDoubleQuoted(s, pos)
		return str, newPos, err
	case pos < len(s) && s[pos] == '\'':
		str, newPos := flowParseSingleQuoted(s, pos)
		return str, newPos, nil
	default:
		start := pos
		for pos < len(s) {
			c := s[pos]
			if c == ':' || c == ',' || c == '}' || c == ']' {
				break
			}
			pos++
		}
		return strings.TrimSpace(s[start:pos]), pos, nil
	}
}

// flowParseDoubleQuoted reads a double-quoted string starting at s[pos].
func flowParseDoubleQuoted(s string, pos int) (string, int, error) {
	end := pos + 1
	for end < len(s) {
		if s[end] == '\\' {
			end += 2
			continue
		}
		if s[end] == '"' {
			str, err := parseDoubleQuoted(s[pos : end+1])
			return str, end + 1, err
		}
		end++
	}
	return "", end, fmt.Errorf("unterminated double-quoted string")
}

// flowParseSingleQuoted reads a single-quoted string starting at s[pos].
func flowParseSingleQuoted(s string, pos int) (string, int) {
	end := pos + 1
	for end < len(s) {
		if s[end] == '\'' {
			if end+1 < len(s) && s[end+1] == '\'' {
				end += 2
				continue
			}
			return parseSingleQuoted(s[pos : end+1]), end + 1
		}
		end++
	}
	return parseSingleQuoted(s[pos:]), end
}

// flowSkipWS advances pos past spaces, tabs, and newlines.
func flowSkipWS(s string, pos int) int {
	for pos < len(s) {
		c := s[pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			pos++
		} else {
			break
		}
	}
	return pos
}
