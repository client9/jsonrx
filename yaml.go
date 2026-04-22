// YAML-to-JSON support. Converts a subset of YAML to JSON without reflection
// or intermediate data structures.
//
// Supported: block mappings, block sequences, flow style, bare/quoted scalars,
// null/bool literals, simple numbers, nested structures, comments, frontmatter,
// literal (|) and folded (>) block scalars.
//
// Not supported: anchors & aliases, tags, complex keys (? ...).

package tojson

import "bytes"

func yamlConvert(input []byte) ([]byte, error) {
	var p parser
	p.init(input)
	if len(p.lines) == 0 {
		return []byte("null"), nil
	}
	var buf bytes.Buffer
	buf.Grow(len(input) + 64)
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
	rawLines [][]byte // original input lines (split on \n, \r stripped)
	rawIdx   []int    // rawIdx[i] = index into rawLines for lines[i]
}

type pline struct {
	indent  int
	content []byte // leading whitespace stripped, trailing whitespace stripped
}

// init populates p from input. Kept as a method so yamlConvert can declare
// parser on the stack and avoid the &parser{} heap escape.
func (p *parser) init(input []byte) {
	// Count newlines for pre-allocation — avoids repeated slice growth.
	n := bytes.Count(input, []byte{'\n'}) + 1

	// Build rawLines without bytes.Split to avoid genSplit's backing alloc;
	// pre-allocate with n so we get exactly one allocation.
	// We must match bytes.Split semantics: always emit one element after the
	// last separator, even when it is empty (so "a\n" → ["a",""] not ["a"]).
	rawLines := make([][]byte, 0, n)
	remaining := input
	for {
		i := bytes.IndexByte(remaining, '\n')
		if i < 0 {
			rawLines = append(rawLines, remaining)
			break
		}
		rawLines = append(rawLines, remaining[:i])
		remaining = remaining[i+1:]
		if len(remaining) == 0 {
			rawLines = append(rawLines, remaining) // trailing empty matches bytes.Split
			break
		}
	}
	// Remove spurious trailing empty element from a \n-terminated input.
	if len(rawLines) > 0 && len(rawLines[len(rawLines)-1]) == 0 {
		rawLines = rawLines[:len(rawLines)-1]
	}

	lines := make([]pline, 0, n)
	rawIdx := make([]int, 0, n)
	for i, raw := range rawLines {
		s := bytes.TrimRight(raw, " \t\r")
		if len(s) == 0 {
			continue
		}
		trimmed := bytes.TrimSpace(s)
		// skip blank, comment-only, and YAML document-marker lines
		if len(trimmed) == 0 || trimmed[0] == '#' ||
			bytes.Equal(trimmed, []byte("---")) || bytes.Equal(trimmed, []byte("...")) {
			continue
		}
		indent := leadingSpaces(s)
		content := s[indent:]
		// strip inline comment (outside quotes) — best-effort
		content = stripInlineComment(content)
		if len(content) == 0 {
			continue
		}
		lines = append(lines, pline{indent: indent, content: content})
		rawIdx = append(rawIdx, i)
	}
	p.lines = lines
	p.rawLines = rawLines
	p.rawIdx = rawIdx
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
// position. Only considers lines with indent > parentIndent, except that
// a block sequence may begin at the same indent as its parent mapping key
// (YAML compact notation).
func (p *parser) parseBlock(parentIndent int, buf *bytes.Buffer) error {
	l, ok := p.peek()
	if !ok {
		buf.WriteString("null")
		return nil
	}
	if l.indent <= parentIndent {
		// Compact notation: block sequence value at same indent as mapping key.
		if l.indent == parentIndent && isSeqItem(l.content) {
			return p.parseSequence(l.indent, buf)
		}
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
				return atLineCol(rawLine, l.indent, err)
			}
			p.skipPastRawLine(last)
			return nil
		}
		if err := writeScalar(l.content, buf); err != nil {
			return atLineCol(rawLine, l.indent, err)
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

		if len(rest) == 0 {
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
				return atLineCol(rawLine, l.indent+len(l.content)-len(rest), err)
			}
			p.skipPastRawLine(last)
		} else {
			if err := writeScalar(rest, buf); err != nil {
				return atLineCol(rawLine, l.indent+len(l.content)-len(rest), err)
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

		rest := bytes.TrimPrefix(l.content, []byte("-"))
		if len(rest) > 0 && rest[0] == ' ' {
			rest = rest[1:]
		}
		rest = bytes.TrimSpace(rest)

		if len(rest) == 0 {
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
				return atLineCol(rawLine, l.indent+len(l.content)-len(rest), err)
			}
			p.skipPastRawLine(last)
		} else {
			if isMapKey(rest) {
				firstLineCol := l.indent + len(l.content) - len(rest)
				if err := p.parseInlineMap(rest, l.indent+2, rawLine, firstLineCol, buf); err != nil {
					return err
				}
			} else {
				if err := writeScalar(rest, buf); err != nil {
					return atLineCol(rawLine, l.indent+len(l.content)-len(rest), err)
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
func (p *parser) parseInlineMap(firstLine []byte, virtIndent int, startRawLine int, firstLineCol int, buf *bytes.Buffer) error {
	buf.WriteByte('{')

	writeKeyValue := func(line []byte, rawLine int, lineCol int) error {
		key, rest := splitMapKey(line)
		writeJSONString(key, buf)
		buf.WriteByte(':')
		if len(rest) == 0 {
			if err := p.parseBlock(virtIndent-1, buf); err != nil {
				return err
			}
		} else if isFlowValue(rest) {
			src, last := p.gatherFlowSrc(rest, rawLine)
			if err := parseFlowExpr(src, buf); err != nil {
				return atLineCol(rawLine, lineCol+len(line)-len(rest), err)
			}
			p.skipPastRawLine(last)
		} else {
			if err := writeScalar(rest, buf); err != nil {
				return atLineCol(rawLine, lineCol+len(line)-len(rest), err)
			}
		}
		return nil
	}

	if err := writeKeyValue(firstLine, startRawLine, firstLineCol); err != nil {
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
		if err := writeKeyValue(l.content, rawLine, l.indent); err != nil {
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
func detectBlockScalar(s []byte) (style, chomping byte, ok bool) {
	s = bytes.TrimSpace(s)
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
func (p *parser) collectBlockScalar(style, chomping byte, rawLineIdx, keyIndent int) ([]byte, int) {
	blockIndent := -1
	var contentLines [][]byte
	lastIdx := rawLineIdx

	for i := rawLineIdx + 1; i < len(p.rawLines); i++ {
		raw := bytes.TrimRight(p.rawLines[i], " \t\r")
		if len(bytes.TrimSpace(raw)) == 0 {
			if blockIndent >= 0 {
				contentLines = append(contentLines, nil)
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

	var result []byte
	if style == '|' {
		result = bytes.Join(contentLines, []byte{'\n'})
	} else {
		var sb bytes.Buffer
		for i, line := range contentLines {
			if len(line) == 0 {
				sb.WriteByte('\n')
			} else {
				if i > 0 && len(contentLines[i-1]) != 0 {
					sb.WriteByte(' ')
				}
				sb.Write(line)
			}
		}
		result = sb.Bytes()
	}

	switch chomping {
	case '-':
		result = bytes.TrimRight(result, "\n")
	case '+':
		if len(contentLines) > 0 {
			result = append(result, '\n')
		}
	default:
		result = bytes.TrimRight(result, "\n")
		if len(contentLines) > 0 {
			result = append(result, '\n')
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
