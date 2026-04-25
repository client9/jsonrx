// toml_line.go — TOML→JSON converter using a top-level line-by-line state machine.
//
// tomlConvertLine is an alternative to tomlConvertStreaming that replaces the
// upfront bytes.Split + per-helper inner-loop design with a single outer loop that
// lazily scans newlines and drives a four-state machine. Multiline constructs
// ("""...""", '''...''', multi-line inline arrays) accumulate content across
// iterations rather than pulling lines from a pre-split slice inside helper functions.
//
// TOML's root is always a table (the spec defines the document as a hash table),
// so the output always begins and ends with { }.

package tojson

import (
	"bytes"
	"fmt"
)

const (
	tomlStateNormal      = iota // reading section headers and key-value pairs
	tomlStateMLBasic            // accumulating a """...""" basic string
	tomlStateMLLiteral          // accumulating a '''...''' literal string
	tomlStateInlineArray        // accumulating a [...] inline array spanning lines
)

// multilineStart reports whether the TOML value s requires more than one line,
// returning the state to enter for accumulation. Returns (false, 0) when the value
// fits on the current line and can be written immediately.
func multilineStart(s []byte) (bool, int) {
	switch {
	case len(s) >= 3 && s[0] == '"' && s[1] == '"' && s[2] == '"':
		if !bytes.Contains(s[3:], []byte(`"""`)) {
			return true, tomlStateMLBasic
		}
	case len(s) >= 3 && s[0] == '\'' && s[1] == '\'' && s[2] == '\'':
		if !bytes.Contains(s[3:], []byte("'''")) {
			return true, tomlStateMLLiteral
		}
	case len(s) > 0 && s[0] == '[':
		if s[len(s)-1] != ']' {
			return true, tomlStateInlineArray
		}
	}
	return false, 0
}

// tomlLineParser holds the mutable state shared across methods during a single
// tomlConvertLine call.
type tomlLineParser struct {
	buf         bytes.Buffer
	stack       []streamFrame
	closed      tomlLineClosedTables
	inlineKeys  [][]byte
	inlineComma []bool
	inlineUsed  [][][]byte
	state       int
	accum       []byte
	startLine   int
	arrayDepth  int
	arrayDouble bool
	arraySingle bool
}

type tomlLineClosedTables struct {
	root tomlLineClosedNode
}

type tomlLineClosedNode struct {
	key      []byte
	closed   bool
	children []tomlLineClosedNode
}

func (n *tomlLineClosedNode) find(key []byte) *tomlLineClosedNode {
	for i := range n.children {
		if bytes.Equal(n.children[i].key, key) {
			return &n.children[i]
		}
	}
	return nil
}

func (n *tomlLineClosedNode) child(key []byte) *tomlLineClosedNode {
	if child := n.find(key); child != nil {
		return child
	}
	n.children = append(n.children, tomlLineClosedNode{key: key})
	return &n.children[len(n.children)-1]
}

func (c *tomlLineClosedTables) mark(stack []streamFrame) {
	if len(stack) <= 1 {
		return
	}
	node := &c.root
	for i := 1; i < len(stack); i++ {
		node = node.child(stack[i].key)
	}
	node.closed = true
}

func (c *tomlLineClosedTables) contains(path [][]byte) bool {
	node := &c.root
	for _, key := range path {
		node = node.find(key)
		if node == nil {
			return false
		}
	}
	return node.closed
}

func (c *tomlLineClosedTables) reopens(path [][]byte, commonDepth int) bool {
	for i := commonDepth; i < len(path); i++ {
		if c.contains(path[:i+1]) {
			return true
		}
	}
	return false
}

func (p *tomlLineParser) topNC() bool {
	if len(p.inlineKeys) > 0 {
		return p.inlineComma[len(p.inlineComma)-1]
	}
	return p.stack[len(p.stack)-1].needComma
}

func (p *tomlLineParser) setTopNC(v bool) {
	if len(p.inlineKeys) > 0 {
		p.inlineComma[len(p.inlineComma)-1] = v
	} else {
		p.stack[len(p.stack)-1].needComma = v
	}
}

func (p *tomlLineParser) markKey(key []byte) error {
	var keys *[][]byte
	if len(p.inlineKeys) > 0 {
		keys = &p.inlineUsed[len(p.inlineUsed)-1]
	} else {
		keys = &p.stack[len(p.stack)-1].usedKeys
	}
	for _, k := range *keys {
		if bytes.Equal(k, key) {
			return fmt.Errorf("duplicate key %q", key)
		}
	}
	*keys = append(*keys, key)
	return nil
}

func (p *tomlLineParser) closeInlineTo(depth int) {
	for len(p.inlineKeys) > depth {
		p.buf.WriteByte('}')
		p.inlineKeys = p.inlineKeys[:len(p.inlineKeys)-1]
		p.inlineComma = p.inlineComma[:len(p.inlineComma)-1]
		p.inlineUsed = p.inlineUsed[:len(p.inlineUsed)-1]
	}
}

func (p *tomlLineParser) closeSectionsTo(depth int) {
	for len(p.stack) > depth {
		top := p.stack[len(p.stack)-1]
		p.closed.mark(p.stack)
		p.stack = p.stack[:len(p.stack)-1]
		if top.isAoT {
			p.buf.WriteString("}]")
		} else {
			p.buf.WriteByte('}')
		}
	}
}

func (p *tomlLineParser) currentSectionIs(path [][]byte) bool {
	if len(path) != len(p.stack)-1 {
		return false
	}
	for i := range path {
		if !bytes.Equal(p.stack[i+1].key, path[i]) {
			return false
		}
	}
	return true
}

func (p *tomlLineParser) openSection(path [][]byte, isAoT bool) error {
	var fullDotPath string
	if len(path) == 1 {
		fullDotPath = string(path[0])
	} else {
		fullDotPath = string(bytes.Join(path, []byte(".")))
	}
	if isAoT && len(p.stack) > 1 {
		top := &p.stack[len(p.stack)-1]
		if top.isAoT && p.currentSectionIs(path) {
			p.buf.WriteString("},{")
			top.needComma = false
			top.usedKeys = top.usedKeys[:0]
			return nil
		}
	}
	cd := 0
	for cd < len(path) && cd+1 < len(p.stack) {
		if !bytes.Equal(p.stack[cd+1].key, path[cd]) {
			break
		}
		cd++
	}
	if p.closed.reopens(path, cd) {
		return errReentry
	}
	p.closeSectionsTo(cd + 1)
	if cd == len(path) {
		frame := &p.stack[len(p.stack)-1]
		if !isAoT {
			if frame.explicit {
				return fmt.Errorf("duplicate table header [%s]", fullDotPath)
			}
			frame.explicit = true
		}
		return nil
	}
	for i := cd; i < len(path); i++ {
		top := &p.stack[len(p.stack)-1]
		for _, k := range top.usedKeys {
			if bytes.Equal(k, path[i]) {
				var dp string
				if i == len(path)-1 {
					dp = fullDotPath
				} else {
					dp = string(bytes.Join(path[:i+1], []byte(".")))
				}
				return fmt.Errorf("cannot define table %q: key already has a value", dp)
			}
		}
		top.usedKeys = append(top.usedKeys, path[i])
		if top.needComma {
			p.buf.WriteByte(',')
		}
		writeJSONString(path[i], &p.buf)
		p.buf.WriteByte(':')
		isAoTFrame := i == len(path)-1 && isAoT
		if isAoTFrame {
			p.buf.WriteString("[{")
		} else {
			p.buf.WriteByte('{')
		}
		top.needComma = true
		var dp string
		if i == len(path)-1 {
			dp = fullDotPath
		} else {
			dp = string(bytes.Join(path[:i+1], []byte(".")))
		}
		p.stack = append(p.stack, streamFrame{
			key:      path[i],
			dotPath:  dp,
			isAoT:    isAoTFrame,
			explicit: i == len(path)-1 && !isAoT,
		})
	}
	return nil
}

// scanArrayLine advances the inline-array parse state by scanning b,
// returning true when the top-level ']' is reached (arrayDepth → 0).
func (p *tomlLineParser) scanArrayLine(b []byte) bool {
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch {
		case p.arrayDouble:
			if c == '\\' {
				i++
			} else if c == '"' {
				p.arrayDouble = false
			}
		case p.arraySingle:
			if c == '\'' {
				p.arraySingle = false
			}
		case c == '"':
			p.arrayDouble = true
		case c == '\'':
			p.arraySingle = true
		case c == '[':
			p.arrayDepth++
		case c == ']':
			p.arrayDepth--
			if p.arrayDepth == 0 {
				return true
			}
		}
	}
	return false
}

func (p *tomlLineParser) appendAccumLine(line []byte) {
	p.accum = append(p.accum, '\n')
	p.accum = append(p.accum, line...)
}

func (p *tomlLineParser) finishAccumValue() {
	p.setTopNC(true)
	p.accum = p.accum[:0]
	p.state = tomlStateNormal
}

func (p *tomlLineParser) handleAccumLine(line []byte) (bool, error) {
	// String content is raw here; stripping comments or whitespace would corrupt
	// multiline values.
	switch p.state {
	case tomlStateNormal:
		return false, nil
	case tomlStateMLBasic:
		p.appendAccumLine(line)
		if !bytes.Contains(line, []byte(`"""`)) {
			return true, nil
		}
		str, _, err := parseTOMLMultilineBasic(p.accum, nil, 0)
		if err != nil {
			return true, atLineCol(p.startLine, 0, err)
		}
		writeJSONString(str, &p.buf)
		p.finishAccumValue()
		return true, nil
	case tomlStateMLLiteral:
		p.appendAccumLine(line)
		if !bytes.Contains(line, []byte("'''")) {
			return true, nil
		}
		str, _, err := parseTOMLMultilineLiteral(p.accum, nil, 0)
		if err != nil {
			return true, atLineCol(p.startLine, 0, err)
		}
		writeJSONString(str, &p.buf)
		p.finishAccumValue()
		return true, nil
	case tomlStateInlineArray:
		p.appendAccumLine(line)
		if !p.scanArrayLine(line) {
			return true, nil
		}
		if _, err := writeTOMLInlineArray(p.accum, nil, 0, &p.buf); err != nil {
			return true, atLineCol(p.startLine, 0, err)
		}
		p.finishAccumValue()
		return true, nil
	default:
		return true, fmt.Errorf("toml: unknown line parser state %d", p.state)
	}
}

func (p *tomlLineParser) handleHeader(trimmed []byte, lineNum, leading int, pathBuf *[4][]byte, isAoT bool) error {
	p.closeInlineTo(0)

	var inner []byte
	if isAoT {
		if !bytes.HasSuffix(trimmed, []byte("]]")) {
			return atLineCol(lineNum, leading, fmt.Errorf("malformed array-of-tables header: %s", trimmed))
		}
		inner = trimmed[2 : len(trimmed)-2]
	} else {
		if trimmed[len(trimmed)-1] != ']' {
			return atLineCol(lineNum, leading, fmt.Errorf("malformed table header: %s", trimmed))
		}
		inner = trimmed[1 : len(trimmed)-1]
	}

	path, rest, err := parseTOMLKeyPath(inner, pathBuf[:0])
	if err != nil {
		return atLineCol(lineNum, leading, err)
	}
	if rest = bytes.TrimSpace(rest); len(rest) != 0 {
		if isAoT {
			return atLineCol(lineNum, leading, fmt.Errorf("unexpected content after [[header]]: %s", rest))
		}
		return atLineCol(lineNum, leading, fmt.Errorf("unexpected content after [header]: %s", rest))
	}
	if err := p.openSection(path, isAoT); err != nil {
		if err == errReentry {
			return err
		}
		return atLineCol(lineNum, leading, err)
	}
	return nil
}

func tomlBareKeyValue(trimmed []byte) (key, rest []byte, ok bool) {
	bareEnd := 0
	for bareEnd < len(trimmed) {
		c := trimmed[bareEnd]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '_' || c == '-' {
			bareEnd++
		} else {
			break
		}
	}
	eqPos := bareEnd
	for eqPos < len(trimmed) && (trimmed[eqPos] == ' ' || trimmed[eqPos] == '\t') {
		eqPos++
	}
	if bareEnd == 0 || eqPos >= len(trimmed) || trimmed[eqPos] != '=' {
		return nil, nil, false
	}
	return trimmed[:bareEnd], bytes.TrimLeft(trimmed[eqPos+1:], " \t"), true
}

func (p *tomlLineParser) handleDottedKeyValue(trimmed []byte, lineNum, leading int, pathBuf *[4][]byte) error {
	path, rest, err := parseTOMLKeyPath(trimmed, pathBuf[:0])
	if err != nil {
		return atLineCol(lineNum, leading, err)
	}
	rest = bytes.TrimSpace(rest)
	if len(rest) == 0 || rest[0] != '=' {
		return atLineCol(lineNum, leading, fmt.Errorf("expected '=' after key, got: %s", rest))
	}
	rest = bytes.TrimSpace(rest[1:])
	valCol := leading + len(trimmed) - len(rest)

	lastKey := path[len(path)-1]
	prefix := path[:len(path)-1]
	if err := p.openInlinePrefix(prefix, lineNum, leading); err != nil {
		return err
	}
	if err := p.markKey(lastKey); err != nil {
		return atLineCol(lineNum, leading, err)
	}
	if p.topNC() {
		p.buf.WriteByte(',')
	}
	writeJSONString(lastKey, &p.buf)
	p.buf.WriteByte(':')
	return p.writeValue(rest, lineNum, valCol)
}

func (p *tomlLineParser) openInlinePrefix(prefix [][]byte, lineNum, leading int) error {
	if len(prefix) == 0 {
		p.closeInlineTo(0)
		return nil
	}

	cd := 0
	for cd < len(prefix) && cd < len(p.inlineKeys) {
		if !bytes.Equal(p.inlineKeys[cd], prefix[cd]) {
			break
		}
		cd++
	}
	p.closeInlineTo(cd)
	for i := cd; i < len(prefix); i++ {
		if err := p.markKey(prefix[i]); err != nil {
			return atLineCol(lineNum, leading, err)
		}
		if p.topNC() {
			p.buf.WriteByte(',')
		}
		writeJSONString(prefix[i], &p.buf)
		p.buf.WriteByte(':')
		p.buf.WriteByte('{')
		p.setTopNC(true)
		p.inlineKeys = append(p.inlineKeys, prefix[i])
		p.inlineComma = append(p.inlineComma, false)
		p.inlineUsed = append(p.inlineUsed, nil)
	}
	return nil
}

func (p *tomlLineParser) startMultilineValue(rest []byte, lineNum, mlState int) {
	p.accum = append(p.accum[:0], rest...)
	p.state = mlState
	p.startLine = lineNum
	if mlState == tomlStateInlineArray {
		p.arrayDepth, p.arrayDouble, p.arraySingle = 0, false, false
		p.scanArrayLine(rest)
	}
}

func (p *tomlLineParser) writeValue(rest []byte, lineNum, valCol int) error {
	if ml, mlState := multilineStart(rest); ml {
		p.startMultilineValue(rest, lineNum, mlState)
		return nil
	}
	if _, err := writeTOMLValue(rest, nil, 0, &p.buf); err != nil {
		return atLineCol(lineNum, valCol, err)
	}
	p.setTopNC(true)
	return nil
}

func tomlConvertLine(input []byte) ([]byte, error) {
	p := &tomlLineParser{
		stack: make([]streamFrame, 1, 8),
	}
	p.buf.Grow(len(input))
	p.buf.WriteByte('{')

	var pathBuf [4][]byte
	lineNum := 0
	pos := 0

	for pos < len(input) {
		// Lazily scan the next line without pre-splitting the whole input.
		nl := bytes.IndexByte(input[pos:], '\n')
		var line []byte
		if nl < 0 {
			line = input[pos:]
			pos = len(input)
		} else {
			line = input[pos : pos+nl]
			pos += nl + 1
		}
		lineNum++

		if handled, err := p.handleAccumLine(line); handled || err != nil {
			if err != nil {
				return nil, err
			}
			continue
		}

		line = bytes.TrimRight(line, " \t\r")
		line = stripInlineComment(line)
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0] == '#' {
			continue
		}
		leading := leadingSpaces(line)

		switch {
		case bytes.HasPrefix(trimmed, []byte("[[")):
			if err := p.handleHeader(trimmed, lineNum, leading, &pathBuf, true); err != nil {
				return nil, err
			}
		case trimmed[0] == '[':
			if err := p.handleHeader(trimmed, lineNum, leading, &pathBuf, false); err != nil {
				return nil, err
			}
		default:
			key, rest, ok := tomlBareKeyValue(trimmed)
			if !ok {
				if err := p.handleDottedKeyValue(trimmed, lineNum, leading, &pathBuf); err != nil {
					return nil, err
				}
				continue
			}
			p.closeInlineTo(0)
			if err := p.markKey(key); err != nil {
				return nil, atLineCol(lineNum, leading, err)
			}
			if p.topNC() {
				p.buf.WriteByte(',')
			}
			writeJSONString(key, &p.buf)
			p.buf.WriteByte(':')
			if ml, mlState := multilineStart(rest); ml {
				p.startMultilineValue(rest, lineNum, mlState)
				continue
			}
			if _, err := writeTOMLValue(rest, nil, 0, &p.buf); err != nil {
				return nil, atLineCol(lineNum, leading+len(trimmed)-len(rest), err)
			}
			p.setTopNC(true)
		}
	}

	if p.state != tomlStateNormal {
		what := "multiline string"
		if p.state == tomlStateInlineArray {
			what = "inline array"
		}
		return nil, atLineCol(p.startLine, 0, fmt.Errorf("unterminated %s", what))
	}

	p.closeInlineTo(0)
	for i := len(p.stack) - 1; i >= 1; i-- {
		if p.stack[i].isAoT {
			p.buf.WriteString("}]")
		} else {
			p.buf.WriteByte('}')
		}
	}
	p.buf.WriteByte('}')
	return p.buf.Bytes(), nil
}

// fromTOMLLine converts TOML to JSON using the line-by-line state-machine path,
// falling back to the tree-based path on out-of-order section re-entry.
func fromTOMLLine(src []byte) ([]byte, error) {
	out, err := tomlConvertLine(src)
	if err == errReentry {
		return tomlConvertTree(src)
	}
	return out, err
}
