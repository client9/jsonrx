// TOML-to-JSON support. Converts a subset of TOML to JSON without reflection
// or intermediate map[string]any structures.
//
// Supported: key-value pairs, standard tables [header], array-of-tables
// [[header]], inline tables {k=v}, inline arrays [v,v], all scalar types,
// dotted keys, comments.
//
// Not supported: TOML integers larger than int64.

package tojson

import (
	"bytes"
	"errors"
	"fmt"
)

// --------------------------------------------------------------------------
// Intermediate JSON node tree
// --------------------------------------------------------------------------

// jnode is a node in the minimal JSON value tree built during TOML parsing.
// Exactly one of raw/obj/arr/aot is non-nil.
type jnode struct {
	raw []byte   // scalar: already-encoded JSON bytes
	obj []*jpair // object: ordered key-value pairs
	arr []*jnode // inline array  (immutable after parse)
	aot []*jnode // array-of-tables (grows with each [[header]])
}

type jpair struct {
	key      []byte
	val      *jnode
	explicit bool // true when created by a [table] header line
}

var (
	nodeTrue  = &jnode{raw: []byte("true")}
	nodeFalse = &jnode{raw: []byte("false")}
)

func newObjectNode() *jnode {
	return &jnode{obj: make([]*jpair, 0, 4)}
}

func newScalarNode(raw []byte) *jnode {
	return &jnode{raw: raw}
}

// findPair returns the jpair with the given key, or nil.
func (n *jnode) findPair(key []byte) *jpair {
	for _, p := range n.obj {
		if bytes.Equal(p.key, key) {
			return p
		}
	}
	return nil
}

// --------------------------------------------------------------------------
// Serializer
// --------------------------------------------------------------------------

func serializeNode(n *jnode, buf *bytes.Buffer) {
	switch {
	case n.raw != nil:
		buf.Write(n.raw)
	case n.obj != nil:
		buf.WriteByte('{')
		for i, p := range n.obj {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeJSONString(p.key, buf)
			buf.WriteByte(':')
			serializeNode(p.val, buf)
		}
		buf.WriteByte('}')
	case n.arr != nil:
		buf.WriteByte('[')
		for i, elem := range n.arr {
			if i > 0 {
				buf.WriteByte(',')
			}
			serializeNode(elem, buf)
		}
		buf.WriteByte(']')
	case n.aot != nil:
		buf.WriteByte('[')
		for i, elem := range n.aot {
			if i > 0 {
				buf.WriteByte(',')
			}
			serializeNode(elem, buf)
		}
		buf.WriteByte(']')
	}
}

// --------------------------------------------------------------------------
// Parser
// --------------------------------------------------------------------------

type tomlParser struct {
	rawLines [][]byte
	lineIdx  int
	root     *jnode
	ctx      *jnode // current table context (reset by [header] and [[header]])
}

func newTOMLParser(input []byte) *tomlParser {
	lines := bytes.Split(input, []byte{'\n'})
	// remove spurious trailing empty element from Split
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	root := newObjectNode()
	return &tomlParser{rawLines: lines, root: root, ctx: root}
}

// errReentry signals that the streaming path detected an out-of-order section
// and the caller should fall back to tomlConvertTree.
var errReentry = errors.New("toml: out-of-order section")

func tomlConvert(input []byte) ([]byte, error) {
	out, err := tomlConvertStreaming(input)
	if err == errReentry {
		return tomlConvertTree(input)
	}
	return out, err
}

func tomlConvertTree(input []byte) ([]byte, error) {
	p := newTOMLParser(input)
	if err := p.parseDocument(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.Grow(len(input))
	serializeNode(p.root, &buf)
	return buf.Bytes(), nil
}

func (p *tomlParser) parseDocument() error {
	for p.lineIdx < len(p.rawLines) {
		line := p.rawLines[p.lineIdx]
		p.lineIdx++
		line = bytes.TrimRight(line, " \t\r")
		line = stripInlineComment(line)
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0] == '#' {
			continue
		}
		leading := leadingSpaces(line)
		if bytes.HasPrefix(trimmed, []byte("[[")) {
			if err := p.parseArrayTableHeader(trimmed); err != nil {
				return atLineCol(p.lineIdx-1, leading, err)
			}
		} else if trimmed[0] == '[' {
			if err := p.parseTableHeader(trimmed); err != nil {
				return atLineCol(p.lineIdx-1, leading, err)
			}
		} else {
			if err := p.parseKeyValue(trimmed, p.lineIdx-1, leading, p.ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

// --------------------------------------------------------------------------
// Table headers
// --------------------------------------------------------------------------

func (p *tomlParser) parseTableHeader(line []byte) error {
	if len(line) < 2 || line[0] != '[' || line[len(line)-1] != ']' {
		return fmt.Errorf("malformed table header: %s", line)
	}
	inner := line[1 : len(line)-1]
	var pathBuf [4][]byte
	path, rest, err := parseTOMLKeyPath(inner, pathBuf[:0])
	if err != nil {
		return err
	}
	rest = bytes.TrimSpace(rest)
	if len(rest) != 0 {
		return fmt.Errorf("unexpected content after table header key: %s", rest)
	}
	if len(path) == 0 {
		return fmt.Errorf("empty table header")
	}
	node, err := p.getOrCreateNode(p.root, path[:len(path)-1], false)
	if err != nil {
		return err
	}
	lastKey := path[len(path)-1]
	existing := node.findPair(lastKey)
	if existing != nil {
		switch {
		case existing.val.raw != nil:
			return fmt.Errorf("cannot define table %q: key already has a scalar value", bytes.Join(path, []byte(".")))
		case existing.val.arr != nil:
			return fmt.Errorf("cannot define table %q: key already has an inline array", bytes.Join(path, []byte(".")))
		case existing.explicit:
			return fmt.Errorf("duplicate table header [%s]", bytes.Join(path, []byte(".")))
		case existing.val.aot != nil:
			// [a] after [[a]] — enter the last aot element
			p.ctx = existing.val.aot[len(existing.val.aot)-1]
			return nil
		}
		// implicit object — mark explicit and use it
		existing.explicit = true
		p.ctx = existing.val
		return nil
	}
	newNode := newObjectNode()
	node.obj = append(node.obj, &jpair{key: lastKey, val: newNode, explicit: true})
	p.ctx = newNode
	return nil
}

func (p *tomlParser) parseArrayTableHeader(line []byte) error {
	if len(line) < 4 || !bytes.HasPrefix(line, []byte("[[")) || !bytes.HasSuffix(line, []byte("]]")) {
		return fmt.Errorf("malformed array-of-tables header: %s", line)
	}
	inner := line[2 : len(line)-2]
	var pathBuf [4][]byte
	path, rest, err := parseTOMLKeyPath(inner, pathBuf[:0])
	if err != nil {
		return err
	}
	rest = bytes.TrimSpace(rest)
	if len(rest) != 0 {
		return fmt.Errorf("unexpected content after array-of-tables header key: %s", rest)
	}
	if len(path) == 0 {
		return fmt.Errorf("empty array-of-tables header")
	}
	node, err := p.getOrCreateNode(p.root, path[:len(path)-1], false)
	if err != nil {
		return err
	}
	lastKey := path[len(path)-1]
	existing := node.findPair(lastKey)
	newEntry := newObjectNode()
	if existing != nil {
		if existing.val.aot == nil {
			return fmt.Errorf("cannot use [[%s]]: key already exists as a non-array", bytes.Join(path, []byte(".")))
		}
		existing.val.aot = append(existing.val.aot, newEntry)
	} else {
		aotNode := &jnode{aot: []*jnode{newEntry}}
		node.obj = append(node.obj, &jpair{key: lastKey, val: aotNode})
	}
	p.ctx = newEntry
	return nil
}

// getOrCreateNode navigates or creates a path of intermediate object nodes
// under root. Used for table headers and dotted key traversal.
func (p *tomlParser) getOrCreateNode(root *jnode, path [][]byte, _ bool) (*jnode, error) {
	cur := root
	for i, key := range path {
		if cur.obj == nil {
			return nil, fmt.Errorf("cannot navigate into non-object node at %q", bytes.Join(path[:i+1], []byte(".")))
		}
		pair := cur.findPair(key)
		if pair == nil {
			next := newObjectNode()
			cur.obj = append(cur.obj, &jpair{key: key, val: next})
			cur = next
			continue
		}
		v := pair.val
		switch {
		case v.raw != nil:
			return nil, fmt.Errorf("key %q already has a scalar value", bytes.Join(path[:i+1], []byte(".")))
		case v.arr != nil:
			return nil, fmt.Errorf("key %q is an inline array and cannot have subtables", bytes.Join(path[:i+1], []byte(".")))
		case v.aot != nil:
			cur = v.aot[len(v.aot)-1]
		default:
			cur = v
		}
	}
	return cur, nil
}

// --------------------------------------------------------------------------
// Key-value parsing
// --------------------------------------------------------------------------

func (p *tomlParser) parseKeyValue(line []byte, rawLine int, leading int, ctx *jnode) error {
	var pathBuf [4][]byte
	path, rest, err := parseTOMLKeyPath(line, pathBuf[:0])
	if err != nil {
		return atLineCol(rawLine, leading, err)
	}
	rest = bytes.TrimSpace(rest)
	if len(rest) == 0 || rest[0] != '=' {
		return atLineCol(rawLine, leading+len(line)-len(rest), fmt.Errorf("expected '=' after key, got: %s", rest))
	}
	rest = bytes.TrimSpace(rest[1:])
	valCol := leading + len(line) - len(rest)

	var targetNode *jnode
	if len(path) > 1 {
		targetNode, err = p.getOrCreateNode(ctx, path[:len(path)-1], false)
		if err != nil {
			return atLineCol(rawLine, leading, err)
		}
	} else {
		targetNode = ctx
	}
	lastKey := path[len(path)-1]
	if targetNode.findPair(lastKey) != nil {
		return atLineCol(rawLine, leading, fmt.Errorf("duplicate key %q", lastKey))
	}

	raw, consumed, err := parseTOMLValue(rest, p.rawLines, p.lineIdx-1)
	if err != nil {
		return atLineCol(rawLine, valCol, err)
	}
	p.lineIdx += consumed

	targetNode.obj = append(targetNode.obj, &jpair{key: lastKey, val: raw})
	return nil
}

// --------------------------------------------------------------------------
// Key path parsing
// --------------------------------------------------------------------------

// parseTOMLKeyPath parses a dotted key (e.g. a."b c".d) from the start of s.
// Returns the decoded key segments and the remainder of s after the last segment.
// buf is caller-provided backing storage (pass yourArray[:0]); avoids a heap alloc for ≤ cap(buf) keys.
func parseTOMLKeyPath(s []byte, buf [][]byte) ([][]byte, []byte, error) {
	keys := buf[:0]
	for {
		s = bytes.TrimLeft(s, " \t")
		if len(s) == 0 {
			break
		}
		var key []byte
		var rest []byte
		var err error
		switch s[0] {
		case '"':
			key, rest, err = parseTOMLBasicStringRaw(s)
			if err != nil {
				return nil, nil, err
			}
		case '\'':
			key, rest = parseTOMLLiteralStringRaw(s)
		default:
			// bare key: [A-Za-z0-9_-]+
			i := 0
			for i < len(s) {
				c := s[i]
				if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
					(c >= '0' && c <= '9') || c == '_' || c == '-' {
					i++
				} else {
					break
				}
			}
			if i == 0 {
				break
			}
			key = s[:i]
			rest = s[i:]
		}
		if len(key) == 0 && len(keys) == 0 {
			break
		}
		keys = append(keys, key)
		rest = bytes.TrimLeft(rest, " \t")
		if len(rest) == 0 || rest[0] != '.' {
			return keys, rest, nil
		}
		s = rest[1:] // consume the dot
	}
	if len(keys) == 0 {
		return nil, s, fmt.Errorf("empty key")
	}
	return keys, s, nil
}

// --------------------------------------------------------------------------
// Streaming translator (single-pass, no intermediate tree)
// --------------------------------------------------------------------------

// streamFrame tracks one open TOML section on the streaming stack.
type streamFrame struct {
	key       []byte
	dotPath   string   // full dot-joined path (string for use as map key)
	isAoT     bool     // opened by [[...]]
	explicit  bool     // set when a [table] header explicitly named this frame
	needComma bool     // next entry in this object needs a leading comma
	usedKeys  [][]byte // lazily allocated; detects duplicate keys via bytes.Equal linear scan
}

// tomlConvertStreaming attempts a single-pass streaming TOML→JSON translation.
// Returns (nil, errReentry) when an out-of-order section header is detected.
func tomlConvertStreaming(input []byte) ([]byte, error) {
	lines := bytes.Split(input, []byte{'\n'})
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}

	var buf bytes.Buffer
	buf.Grow(len(input))
	buf.WriteByte('{')

	stack := make([]streamFrame, 1, 8)
	closed := make(map[string]struct{}, 4)

	var inlineKeys [][]byte
	var inlineComma []bool
	var inlineUsed [][][]byte

	topNC := func() bool {
		if len(inlineKeys) > 0 {
			return inlineComma[len(inlineComma)-1]
		}
		return stack[len(stack)-1].needComma
	}
	setTopNC := func(v bool) {
		if len(inlineKeys) > 0 {
			inlineComma[len(inlineComma)-1] = v
		} else {
			stack[len(stack)-1].needComma = v
		}
	}
	markKey := func(key []byte) error {
		var keys *[][]byte
		if len(inlineKeys) > 0 {
			keys = &inlineUsed[len(inlineUsed)-1]
		} else {
			keys = &stack[len(stack)-1].usedKeys
		}
		for _, k := range *keys {
			if bytes.Equal(k, key) {
				return fmt.Errorf("duplicate key %q", key)
			}
		}
		*keys = append(*keys, key)
		return nil
	}
	closeInlineTo := func(depth int) {
		for len(inlineKeys) > depth {
			buf.WriteByte('}')
			inlineKeys = inlineKeys[:len(inlineKeys)-1]
			inlineComma = inlineComma[:len(inlineComma)-1]
			inlineUsed = inlineUsed[:len(inlineUsed)-1]
		}
	}
	closeSectionsTo := func(depth int) {
		for len(stack) > depth {
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if top.dotPath != "" {
				closed[top.dotPath] = struct{}{}
			}
			if top.isAoT {
				buf.WriteString("}]")
			} else {
				buf.WriteByte('}')
			}
		}
	}

	openSection := func(path [][]byte, isAoT bool) error {
		var fullDotPath string
		if len(path) == 1 {
			fullDotPath = string(path[0])
		} else {
			fullDotPath = string(bytes.Join(path, []byte(".")))
		}

		if isAoT && len(stack) > 1 {
			top := &stack[len(stack)-1]
			if top.isAoT && top.dotPath == fullDotPath {
				buf.WriteString("},{")
				top.needComma = false
				top.usedKeys = top.usedKeys[:0]
				return nil
			}
		}

		cd := 0
		for cd < len(path) && cd+1 < len(stack) {
			if !bytes.Equal(stack[cd+1].key, path[cd]) {
				break
			}
			cd++
		}

		for i := cd; i < len(path); i++ {
			var dp string
			if i == len(path)-1 {
				dp = fullDotPath
			} else {
				dp = string(bytes.Join(path[:i+1], []byte(".")))
			}
			if _, exists := closed[dp]; exists {
				return errReentry
			}
		}

		closeSectionsTo(cd + 1)

		if cd == len(path) {
			frame := &stack[len(stack)-1]
			if !isAoT {
				if frame.explicit {
					return fmt.Errorf("duplicate table header [%s]", fullDotPath)
				}
				frame.explicit = true
			}
			return nil
		}

		for i := cd; i < len(path); i++ {
			top := &stack[len(stack)-1]
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
				buf.WriteByte(',')
			}
			writeJSONString(path[i], &buf)
			buf.WriteByte(':')
			isAoTFrame := i == len(path)-1 && isAoT
			if isAoTFrame {
				buf.WriteString("[{")
			} else {
				buf.WriteByte('{')
			}
			top.needComma = true
			var dp string
			if i == len(path)-1 {
				dp = fullDotPath
			} else {
				dp = string(bytes.Join(path[:i+1], []byte(".")))
			}
			stack = append(stack, streamFrame{
				key:       path[i],
				dotPath:   dp,
				isAoT:     isAoTFrame,
				explicit:  i == len(path)-1 && !isAoT,
				needComma: false,
			})
		}
		return nil
	}

	var pathBuf [4][]byte
	lineIdx := 0
	for lineIdx < len(lines) {
		line := lines[lineIdx]
		lineIdx++
		line = bytes.TrimRight(line, " \t\r")
		line = stripInlineComment(line)
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0] == '#' {
			continue
		}
		leading := leadingSpaces(line)

		if bytes.HasPrefix(trimmed, []byte("[[")) {
			closeInlineTo(0)
			if !bytes.HasSuffix(trimmed, []byte("]]")) {
				return nil, atLineCol(lineIdx-1, leading, fmt.Errorf("malformed array-of-tables header: %s", trimmed))
			}
			path, rest, err := parseTOMLKeyPath(trimmed[2:len(trimmed)-2], pathBuf[:0])
			if err != nil {
				return nil, atLineCol(lineIdx-1, leading, err)
			}
			if rest = bytes.TrimSpace(rest); len(rest) != 0 {
				return nil, atLineCol(lineIdx-1, leading, fmt.Errorf("unexpected content after [[header]]: %s", rest))
			}
			if err := openSection(path, true); err != nil {
				if err == errReentry {
					return nil, err
				}
				return nil, atLineCol(lineIdx-1, leading, err)
			}
		} else if trimmed[0] == '[' {
			closeInlineTo(0)
			if trimmed[len(trimmed)-1] != ']' {
				return nil, atLineCol(lineIdx-1, leading, fmt.Errorf("malformed table header: %s", trimmed))
			}
			path, rest, err := parseTOMLKeyPath(trimmed[1:len(trimmed)-1], pathBuf[:0])
			if err != nil {
				return nil, atLineCol(lineIdx-1, leading, err)
			}
			if rest = bytes.TrimSpace(rest); len(rest) != 0 {
				return nil, atLineCol(lineIdx-1, leading, fmt.Errorf("unexpected content after [header]: %s", rest))
			}
			if err := openSection(path, false); err != nil {
				if err == errReentry {
					return nil, err
				}
				return nil, atLineCol(lineIdx-1, leading, err)
			}
		} else {
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

			if bareEnd > 0 && eqPos < len(trimmed) && trimmed[eqPos] == '=' {
				key := trimmed[:bareEnd]
				rest := bytes.TrimLeft(trimmed[eqPos+1:], " \t")
				closeInlineTo(0)
				if err := markKey(key); err != nil {
					return nil, atLineCol(lineIdx-1, leading, err)
				}
				if topNC() {
					buf.WriteByte(',')
				}
				writeJSONString(key, &buf)
				buf.WriteByte(':')
				consumed, err := writeTOMLValue(rest, lines, lineIdx-1, &buf)
				if err != nil {
					return nil, atLineCol(lineIdx-1, leading+len(trimmed)-len(rest), err)
				}
				lineIdx += consumed
				setTopNC(true)
			} else {
				path, rest, err := parseTOMLKeyPath(trimmed, pathBuf[:0])
				if err != nil {
					return nil, atLineCol(lineIdx-1, leading, err)
				}
				rest = bytes.TrimSpace(rest)
				if len(rest) == 0 || rest[0] != '=' {
					return nil, atLineCol(lineIdx-1, leading, fmt.Errorf("expected '=' after key, got: %s", rest))
				}
				rest = bytes.TrimSpace(rest[1:])
				valCol := leading + len(trimmed) - len(rest)

				lastKey := path[len(path)-1]
				prefix := path[:len(path)-1]

				if len(prefix) > 0 {
					cd := 0
					for cd < len(prefix) && cd < len(inlineKeys) {
						if !bytes.Equal(inlineKeys[cd], prefix[cd]) {
							break
						}
						cd++
					}
					closeInlineTo(cd)
					for i := cd; i < len(prefix); i++ {
						if err := markKey(prefix[i]); err != nil {
							return nil, atLineCol(lineIdx-1, leading, err)
						}
						if topNC() {
							buf.WriteByte(',')
						}
						writeJSONString(prefix[i], &buf)
						buf.WriteByte(':')
						buf.WriteByte('{')
						setTopNC(true)
						inlineKeys = append(inlineKeys, prefix[i])
						inlineComma = append(inlineComma, false)
						inlineUsed = append(inlineUsed, nil)
					}
				} else {
					closeInlineTo(0)
				}

				if err := markKey(lastKey); err != nil {
					return nil, atLineCol(lineIdx-1, leading, err)
				}
				if topNC() {
					buf.WriteByte(',')
				}
				writeJSONString(lastKey, &buf)
				buf.WriteByte(':')
				consumed, err := writeTOMLValue(rest, lines, lineIdx-1, &buf)
				if err != nil {
					return nil, atLineCol(lineIdx-1, valCol, err)
				}
				lineIdx += consumed
				setTopNC(true)
			}
		}
	}

	closeInlineTo(0)
	for i := len(stack) - 1; i >= 1; i-- {
		if stack[i].isAoT {
			buf.WriteString("}]")
		} else {
			buf.WriteByte('}')
		}
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}
