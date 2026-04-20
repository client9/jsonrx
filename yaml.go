// YAML-to-JSON support. Converts a subset of YAML to JSON without reflection
// or intermediate data structures.
//
// Supported: block mappings, block sequences, flow style, bare/quoted scalars,
// null/bool literals, simple numbers, nested structures, comments, frontmatter,
// literal (|) and folded (>) block scalars.
//
// Not supported: anchors & aliases, tags, complex keys (? ...).

package tojson

import (
	"bytes"
	"fmt"
	"strings"
)

// ParseError is returned by FromYAML when the input cannot be parsed.
type ParseError struct {
	Line int    // 1-based line number in the original input
	Msg  string // description of the problem
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Msg)
}

// atLine wraps err with a 1-based line number unless it is already a ParseError.
// rawLine is a 0-based index into the original rawLines slice.
func atLine(rawLine int, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*ParseError); ok {
		return err
	}
	return &ParseError{Line: rawLine + 1, Msg: err.Error()}
}

func yamlConvert(input string) ([]byte, error) {
	p := newParser(input)
	if len(p.lines) == 0 {
		return []byte("null"), nil
	}
	var buf bytes.Buffer
	if err := p.parseBlock(-1, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// --------------------------------------------------------------------------
// Parser
// --------------------------------------------------------------------------

type parser struct {
	lines    []pline
	pos      int
	rawLines []string // original input lines (split on \n, \r stripped)
	rawIdx   []int    // rawIdx[i] = index into rawLines for lines[i]
}

type pline struct {
	indent  int
	content string // leading whitespace stripped, trailing whitespace stripped
}

func newParser(input string) *parser {
	rawLines := strings.Split(input, "\n")
	// strings.Split on a \n-terminated string adds a spurious trailing empty
	// element; remove it so it isn't mistaken for a blank line in block scalars.
	if len(rawLines) > 0 && rawLines[len(rawLines)-1] == "" {
		rawLines = rawLines[:len(rawLines)-1]
	}
	var lines []pline
	var rawIdx []int
	for i, raw := range rawLines {
		s := strings.TrimRight(raw, " \t\r")
		if s == "" {
			continue
		}
		trimmed := strings.TrimSpace(s)
		// skip blank, comment-only, and YAML document-marker lines
		if trimmed == "" || strings.HasPrefix(trimmed, "#") ||
			trimmed == "---" || trimmed == "..." {
			continue
		}
		indent := leadingSpaces(s)
		content := s[indent:]
		// strip inline comment (outside quotes) — best-effort
		content = stripInlineComment(content)
		if content == "" {
			continue
		}
		lines = append(lines, pline{indent: indent, content: content})
		rawIdx = append(rawIdx, i)
	}
	return &parser{lines: lines, rawLines: rawLines, rawIdx: rawIdx}
}

func (p *parser) peek() (pline, bool) {
	if p.pos >= len(p.lines) {
		return pline{}, false
	}
	return p.lines[p.pos], true
}

func (p *parser) consume() pline {
	l := p.lines[p.pos]
	p.pos++
	return l
}

// parseBlock writes a JSON value for the block starting at the current
// position. Only considers lines with indent > parentIndent.
func (p *parser) parseBlock(parentIndent int, buf *bytes.Buffer) error {
	l, ok := p.peek()
	if !ok || l.indent <= parentIndent {
		buf.WriteString("null")
		return nil
	}
	blockIndent := l.indent

	switch {
	case isSeqItem(l.content):
		return p.parseSequence(blockIndent, buf)
	case isMapKey(l.content):
		return p.parseMapping(blockIndent, buf)
	default:
		p.consume()
		rawLine := p.rawIdx[p.pos-1]
		if style, chomping, ok := detectBlockScalar(l.content); ok {
			scalar, last := p.collectBlockScalar(style, chomping, rawLine, l.indent)
			p.skipPastRawLine(last)
			writeJSONString(scalar, buf)
			return nil
		}
		if isFlowValue(l.content) {
			src, last := p.gatherFlowSrc(l.content, rawLine)
			if err := parseFlowExpr(src, buf); err != nil {
				return atLine(rawLine, err)
			}
			p.skipPastRawLine(last)
			return nil
		}
		if err := writeScalar(l.content, buf); err != nil {
			return atLine(rawLine, err)
		}
		return nil
	}
}

// parseMapping writes a JSON object for all map-key lines at indent.
func (p *parser) parseMapping(indent int, buf *bytes.Buffer) error {
	buf.WriteByte('{')
	first := true
	for {
		l, ok := p.peek()
		if !ok || l.indent != indent || !isMapKey(l.content) {
			break
		}
		if !first {
			buf.WriteByte(',')
		}
		first = false
		p.consume()
		rawLine := p.rawIdx[p.pos-1]

		key, rest := splitMapKey(l.content)
		writeJSONString(key, buf)
		buf.WriteByte(':')

		if rest == "" {
			if err := p.parseBlock(indent, buf); err != nil {
				return err
			}
		} else if style, chomping, ok := detectBlockScalar(rest); ok {
			scalar, last := p.collectBlockScalar(style, chomping, rawLine, l.indent)
			p.skipPastRawLine(last)
			writeJSONString(scalar, buf)
		} else if isFlowValue(rest) {
			src, last := p.gatherFlowSrc(rest, rawLine)
			if err := parseFlowExpr(src, buf); err != nil {
				return atLine(rawLine, err)
			}
			p.skipPastRawLine(last)
		} else {
			if err := writeScalar(rest, buf); err != nil {
				return atLine(rawLine, err)
			}
		}
	}
	buf.WriteByte('}')
	return nil
}

// parseSequence writes a JSON array for all sequence-item lines at indent.
func (p *parser) parseSequence(indent int, buf *bytes.Buffer) error {
	buf.WriteByte('[')
	first := true
	for {
		l, ok := p.peek()
		if !ok || l.indent != indent || !isSeqItem(l.content) {
			break
		}
		if !first {
			buf.WriteByte(',')
		}
		first = false
		p.consume()
		rawLine := p.rawIdx[p.pos-1]

		rest := strings.TrimPrefix(l.content, "-")
		if len(rest) > 0 && rest[0] == ' ' {
			rest = rest[1:]
		}
		rest = strings.TrimSpace(rest)

		if rest == "" {
			if err := p.parseBlock(indent, buf); err != nil {
				return err
			}
		} else if style, chomping, ok := detectBlockScalar(rest); ok {
			scalar, last := p.collectBlockScalar(style, chomping, rawLine, l.indent)
			p.skipPastRawLine(last)
			writeJSONString(scalar, buf)
		} else if isFlowValue(rest) {
			src, last := p.gatherFlowSrc(rest, rawLine)
			if err := parseFlowExpr(src, buf); err != nil {
				return atLine(rawLine, err)
			}
			p.skipPastRawLine(last)
		} else {
			if isMapKey(rest) {
				if err := p.parseInlineMap(rest, l.indent+2, rawLine, buf); err != nil {
					return err
				}
			} else {
				if err := writeScalar(rest, buf); err != nil {
					return atLine(rawLine, err)
				}
			}
		}
	}
	buf.WriteByte(']')
	return nil
}

// parseInlineMap handles the case where a sequence item starts an inline
// mapping on the same line as the dash, e.g.:
//
//   - name: Alice
//     age: 30
func (p *parser) parseInlineMap(firstLine string, virtIndent int, startRawLine int, buf *bytes.Buffer) error {
	buf.WriteByte('{')

	writeKeyValue := func(line string, rawLine int) error {
		key, rest := splitMapKey(line)
		writeJSONString(key, buf)
		buf.WriteByte(':')
		if rest == "" {
			if err := p.parseBlock(virtIndent-1, buf); err != nil {
				return err
			}
		} else if isFlowValue(rest) {
			src, last := p.gatherFlowSrc(rest, rawLine)
			if err := parseFlowExpr(src, buf); err != nil {
				return atLine(rawLine, err)
			}
			p.skipPastRawLine(last)
		} else {
			if err := writeScalar(rest, buf); err != nil {
				return atLine(rawLine, err)
			}
		}
		return nil
	}

	if err := writeKeyValue(firstLine, startRawLine); err != nil {
		return err
	}

	for {
		l, ok := p.peek()
		if !ok || l.indent != virtIndent || !isMapKey(l.content) {
			break
		}
		buf.WriteByte(',')
		p.consume()
		rawLine := p.rawIdx[p.pos-1]
		if err := writeKeyValue(l.content, rawLine); err != nil {
			return err
		}
	}

	buf.WriteByte('}')
	return nil
}

// --------------------------------------------------------------------------
// Block scalar support (| and >)
// --------------------------------------------------------------------------

// detectBlockScalar returns the style ('|' or '>'), chomping ('-' strip,
// '+' keep, 0 clip/default), and ok=true if s is a block scalar indicator.
func detectBlockScalar(s string) (style, chomping byte, ok bool) {
	s = strings.TrimSpace(s)
	if len(s) == 0 || (s[0] != '|' && s[0] != '>') {
		return 0, 0, false
	}
	style = s[0]
	for _, c := range s[1:] {
		switch c {
		case '-':
			chomping = '-'
		case '+':
			chomping = '+'
		case '1', '2', '3', '4', '5', '6', '7', '8', '9':
			// explicit indentation indicator — ignored, auto-detect instead
		default:
			return 0, 0, false
		}
	}
	return style, chomping, true
}

// collectBlockScalar reads raw lines following rawLineIdx to build a literal
// (style='|') or folded (style='>') scalar.
func (p *parser) collectBlockScalar(style, chomping byte, rawLineIdx, keyIndent int) (string, int) {
	blockIndent := -1
	var contentLines []string
	lastIdx := rawLineIdx

	for i := rawLineIdx + 1; i < len(p.rawLines); i++ {
		raw := strings.TrimRight(p.rawLines[i], " \t\r")
		if strings.TrimSpace(raw) == "" {
			if blockIndent >= 0 {
				contentLines = append(contentLines, "")
				lastIdx = i
			}
			continue
		}
		ind := leadingSpaces(raw)
		if blockIndent < 0 {
			if ind <= keyIndent {
				break
			}
			blockIndent = ind
		}
		if ind < blockIndent {
			break
		}
		contentLines = append(contentLines, raw[blockIndent:])
		lastIdx = i
	}

	var result string
	if style == '|' {
		result = strings.Join(contentLines, "\n")
	} else {
		var sb strings.Builder
		for i, line := range contentLines {
			if line == "" {
				sb.WriteByte('\n')
			} else {
				if i > 0 && contentLines[i-1] != "" {
					sb.WriteByte(' ')
				}
				sb.WriteString(line)
			}
		}
		result = sb.String()
	}

	switch chomping {
	case '-':
		result = strings.TrimRight(result, "\n")
	case '+':
		if len(contentLines) > 0 {
			result += "\n"
		}
	default:
		result = strings.TrimRight(result, "\n")
		if len(contentLines) > 0 {
			result += "\n"
		}
	}

	return result, lastIdx
}

// skipPastRawLine advances p.pos past all plines whose raw-line index is ≤ lastRawIdx.
func (p *parser) skipPastRawLine(lastRawIdx int) {
	for p.pos < len(p.lines) && p.rawIdx[p.pos] <= lastRawIdx {
		p.pos++
	}
}
