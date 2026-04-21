package tojson

import (
	"bytes"
	"fmt"
)

// isFlowValue reports whether s begins with a flow mapping or sequence.
func isFlowValue(s []byte) bool {
	s = bytes.TrimSpace(s)
	return len(s) > 0 && (s[0] == '{' || s[0] == '[')
}

// flowDepth returns the net count of open flow delimiters minus closed ones,
// ignoring content inside quoted strings.
func flowDepth(s []byte) int {
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

// gatherFlowSrc builds a complete flow expression starting with first.
// If brackets are unbalanced it reads additional rawLines to support multi-line
// flow values. Returns the assembled bytes and the last rawLine index consumed.
func (p *parser) gatherFlowSrc(first []byte, rawLineIdx int) ([]byte, int) {
	var sb bytes.Buffer
	sb.Write(first)
	depth := flowDepth(first)
	last := rawLineIdx
	for depth > 0 {
		rawLineIdx++
		if rawLineIdx >= len(p.rawLines) {
			break
		}
		line := bytes.TrimRight(p.rawLines[rawLineIdx], " \t\r")
		line = stripInlineComment(bytes.TrimSpace(line))
		if len(line) == 0 {
			continue
		}
		sb.WriteByte(' ')
		sb.Write(line)
		depth += flowDepth(line)
		last = rawLineIdx
	}
	return sb.Bytes(), last
}

// parseFlowExpr parses a complete YAML flow expression (mapping, sequence, or
// scalar) from s and writes its JSON representation to buf.
func parseFlowExpr(s []byte, buf *bytes.Buffer) error {
	s = bytes.TrimSpace(s)
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
func parseFlowMapping(s []byte, pos int, buf *bytes.Buffer) (int, error) {
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
func parseFlowSequence(s []byte, pos int, buf *bytes.Buffer) (int, error) {
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
func flowParseItem(s []byte, pos int, buf *bytes.Buffer) (int, error) {
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
		return pos, writeScalar(bytes.TrimSpace(s[start:pos]), buf)
	}
}

// flowParseKey reads a mapping key (bare, double-quoted, or single-quoted).
func flowParseKey(s []byte, pos int) ([]byte, int, error) {
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
		return bytes.TrimSpace(s[start:pos]), pos, nil
	}
}

// flowParseDoubleQuoted reads a double-quoted string starting at s[pos].
func flowParseDoubleQuoted(s []byte, pos int) ([]byte, int, error) {
	str, end, err := parseDoubleQuoted(s[pos:])
	return str, pos + end, err
}

// flowParseSingleQuoted reads a single-quoted string starting at s[pos].
func flowParseSingleQuoted(s []byte, pos int) ([]byte, int) {
	str, rest := parseSingleQuotedRaw(s[pos:])
	return str, len(s) - len(rest)
}

// flowSkipWS advances pos past spaces, tabs, and newlines.
func flowSkipWS(s []byte, pos int) int {
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
