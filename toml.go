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
	"strings"
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
	key      string
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
func (n *jnode) findPair(key string) *jpair {
	for _, p := range n.obj {
		if p.key == key {
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
	rawLines []string
	lineIdx  int
	root     *jnode
	ctx      *jnode // current table context (reset by [header] and [[header]])
}

func newTOMLParser(input string) *tomlParser {
	lines := strings.Split(input, "\n")
	// remove spurious trailing empty element from Split
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	root := newObjectNode()
	return &tomlParser{rawLines: lines, root: root, ctx: root}
}

// errReentry signals that the streaming path detected an out-of-order section
// and the caller should fall back to tomlConvertTree.
var errReentry = errors.New("toml: out-of-order section")

func tomlConvert(input string) ([]byte, error) {
	out, err := tomlConvertStreaming(input)
	if err == errReentry {
		return tomlConvertTree(input)
	}
	return out, err
}

func tomlConvertTree(input string) ([]byte, error) {
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
		// strip trailing whitespace and CR
		line = strings.TrimRight(line, " \t\r")
		// strip inline comment
		line = stripInlineComment(line)
		// trim leading whitespace
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[[") {
			if err := p.parseArrayTableHeader(trimmed); err != nil {
				return atLine(p.lineIdx-1, err)
			}
		} else if trimmed[0] == '[' {
			if err := p.parseTableHeader(trimmed); err != nil {
				return atLine(p.lineIdx-1, err)
			}
		} else {
			if err := p.parseKeyValue(trimmed, p.ctx); err != nil {
				return atLine(p.lineIdx-1, err)
			}
		}
	}
	return nil
}

// --------------------------------------------------------------------------
// Table headers
// --------------------------------------------------------------------------

func (p *tomlParser) parseTableHeader(line string) error {
	// line is trimmed, starts with '['
	if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
		return fmt.Errorf("malformed table header: %s", line)
	}
	inner := line[1 : len(line)-1]
	path, rest, err := parseTOMLKeyPath(inner)
	if err != nil {
		return err
	}
	rest = strings.TrimSpace(rest)
	if rest != "" {
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
			return fmt.Errorf("cannot define table %q: key already has a scalar value", strings.Join(path, "."))
		case existing.val.arr != nil:
			return fmt.Errorf("cannot define table %q: key already has an inline array", strings.Join(path, "."))
		case existing.explicit:
			return fmt.Errorf("duplicate table header [%s]", strings.Join(path, "."))
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

func (p *tomlParser) parseArrayTableHeader(line string) error {
	// line is trimmed, must start with '[[' and end with ']]'
	if !strings.HasPrefix(line, "[[") || !strings.HasSuffix(line, "]]") {
		return fmt.Errorf("malformed array-of-tables header: %s", line)
	}
	inner := line[2 : len(line)-2]
	path, rest, err := parseTOMLKeyPath(inner)
	if err != nil {
		return err
	}
	rest = strings.TrimSpace(rest)
	if rest != "" {
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
			return fmt.Errorf("cannot use [[%s]]: key already exists as a non-array", strings.Join(path, "."))
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
// forAoT=true means we're about to append a new [[aot]] entry (only used
// internally when the last segment is the AoT itself).
func (p *tomlParser) getOrCreateNode(root *jnode, path []string, _ bool) (*jnode, error) {
	cur := root
	for i, key := range path {
		if cur.obj == nil {
			return nil, fmt.Errorf("cannot navigate into non-object node at %q", strings.Join(path[:i+1], "."))
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
			return nil, fmt.Errorf("key %q already has a scalar value", strings.Join(path[:i+1], "."))
		case v.arr != nil:
			return nil, fmt.Errorf("key %q is an inline array and cannot have subtables", strings.Join(path[:i+1], "."))
		case v.aot != nil:
			// navigate into the last (current) aot entry
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

func (p *tomlParser) parseKeyValue(line string, ctx *jnode) error {
	path, rest, err := parseTOMLKeyPath(line)
	if err != nil {
		return err
	}
	rest = strings.TrimSpace(rest)
	if !strings.HasPrefix(rest, "=") {
		return fmt.Errorf("expected '=' after key, got: %s", rest)
	}
	rest = strings.TrimSpace(rest[1:])

	// navigate/create intermediate nodes for dotted keys
	var targetNode *jnode
	if len(path) > 1 {
		targetNode, err = p.getOrCreateNode(ctx, path[:len(path)-1], false)
		if err != nil {
			return err
		}
	} else {
		targetNode = ctx
	}
	lastKey := path[len(path)-1]
	if targetNode.findPair(lastKey) != nil {
		return fmt.Errorf("duplicate key %q", lastKey)
	}

	raw, consumed, err := parseTOMLValue(rest, p.rawLines, p.lineIdx-1)
	if err != nil {
		return err
	}
	// advance lineIdx for multiline values
	p.lineIdx += consumed

	targetNode.obj = append(targetNode.obj, &jpair{key: lastKey, val: raw})
	return nil
}

// --------------------------------------------------------------------------
// Key path parsing
// --------------------------------------------------------------------------

// parseTOMLKeyPath parses a dotted key (e.g. a."b c".d) from the start of s.
// Returns the decoded key segments and the remainder of s after the last segment.
func parseTOMLKeyPath(s string) ([]string, string, error) {
	var keysBuf [4]string
	keys := keysBuf[:0]
	for {
		s = strings.TrimLeft(s, " \t")
		if len(s) == 0 {
			break
		}
		var key string
		var rest string
		var err error
		switch s[0] {
		case '"':
			key, rest, err = parseTOMLBasicStringRaw(s)
			if err != nil {
				return nil, "", err
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
		if key == "" && len(keys) == 0 {
			break
		}
		keys = append(keys, key)
		rest = strings.TrimLeft(rest, " \t")
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
	key       string
	dotPath   string              // full dot-joined path to this frame
	isAoT     bool                // opened by [[...]]
	explicit  bool                // set when a [table] header explicitly named this frame
	needComma bool                // next entry in this object needs a leading comma
	usedKeys  map[string]struct{} // lazily allocated; detects duplicate keys
}

// tomlConvertStreaming attempts a single-pass streaming TOML→JSON translation.
// Returns (nil, errReentry) when an out-of-order section header is detected.
func tomlConvertStreaming(input string) ([]byte, error) {
	lines := strings.Split(input, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	var buf bytes.Buffer
	buf.Grow(len(input))
	buf.WriteByte('{')

	// stack[0] is the implicit root object.
	stack := make([]streamFrame, 1, 8)
	// closed records dot-paths of sections that have been sealed.
	closed := make(map[string]struct{}, 4)

	// Inline dotted-key tracking within the current section.
	var inlineKeys []string
	var inlineComma []bool
	var inlineUsed []map[string]struct{}

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
	// markKey checks for duplicates and records key in the current scope's used set.
	markKey := func(key string) error {
		var m map[string]struct{}
		if len(inlineKeys) > 0 {
			m = inlineUsed[len(inlineUsed)-1]
			if m == nil {
				m = make(map[string]struct{}, 4)
				inlineUsed[len(inlineUsed)-1] = m
			}
		} else {
			f := &stack[len(stack)-1]
			if f.usedKeys == nil {
				f.usedKeys = make(map[string]struct{}, 4)
			}
			m = f.usedKeys
		}
		if _, dup := m[key]; dup {
			return fmt.Errorf("duplicate key %q", key)
		}
		m[key] = struct{}{}
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

	// openSection handles a [path] or [[path]] header line.
	openSection := func(path []string, isAoT bool) error {
		// Precompute the full dot-joined path once.
		var fullDotPath string
		if len(path) == 1 {
			fullDotPath = path[0]
		} else {
			fullDotPath = strings.Join(path, ".")
		}

		// AoT append: [[same.path]] matching the current top-of-stack AoT.
		if isAoT && len(stack) > 1 {
			top := &stack[len(stack)-1]
			if top.isAoT && top.dotPath == fullDotPath {
				buf.WriteString("},{")
				top.needComma = false
				top.usedKeys = nil // new element — reset key tracking
				return nil
			}
		}

		// Compute depth at which new path diverges from current stack.
		cd := 0
		for cd < len(path) && cd+1 < len(stack) {
			if stack[cd+1].key != path[cd] {
				break
			}
			cd++
		}

		// If any new segment was already closed, bail.
		for i := cd; i < len(path); i++ {
			var dp string
			if i == len(path)-1 {
				dp = fullDotPath
			} else {
				dp = strings.Join(path[:i+1], ".")
			}
			if _, exists := closed[dp]; exists {
				return errReentry
			}
		}

		// Seal sections beyond the common depth.
		closeSectionsTo(cd + 1)

		// Exact match: re-entering an already-open section.
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

		// Open new frames for each new path segment.
		for i := cd; i < len(path); i++ {
			top := &stack[len(stack)-1]
			if top.usedKeys != nil {
				if _, dup := top.usedKeys[path[i]]; dup {
					var dp string
					if i == len(path)-1 {
						dp = fullDotPath
					} else {
						dp = strings.Join(path[:i+1], ".")
					}
					return fmt.Errorf("cannot define table %q: key already has a value", dp)
				}
			}
			if top.usedKeys == nil {
				top.usedKeys = make(map[string]struct{}, 4)
			}
			top.usedKeys[path[i]] = struct{}{}

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
				dp = strings.Join(path[:i+1], ".")
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

	lineIdx := 0
	for lineIdx < len(lines) {
		line := lines[lineIdx]
		lineIdx++
		line = strings.TrimRight(line, " \t\r")
		line = stripInlineComment(line)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if strings.HasPrefix(trimmed, "[[") {
			closeInlineTo(0)
			if !strings.HasSuffix(trimmed, "]]") {
				return nil, atLine(lineIdx-1, fmt.Errorf("malformed array-of-tables header: %s", trimmed))
			}
			path, rest, err := parseTOMLKeyPath(trimmed[2 : len(trimmed)-2])
			if err != nil {
				return nil, atLine(lineIdx-1, err)
			}
			if rest = strings.TrimSpace(rest); rest != "" {
				return nil, atLine(lineIdx-1, fmt.Errorf("unexpected content after [[header]]: %s", rest))
			}
			if err := openSection(path, true); err != nil {
				if err == errReentry {
					return nil, err
				}
				return nil, atLine(lineIdx-1, err)
			}
		} else if trimmed[0] == '[' {
			closeInlineTo(0)
			if !strings.HasSuffix(trimmed, "]") {
				return nil, atLine(lineIdx-1, fmt.Errorf("malformed table header: %s", trimmed))
			}
			path, rest, err := parseTOMLKeyPath(trimmed[1 : len(trimmed)-1])
			if err != nil {
				return nil, atLine(lineIdx-1, err)
			}
			if rest = strings.TrimSpace(rest); rest != "" {
				return nil, atLine(lineIdx-1, fmt.Errorf("unexpected content after [header]: %s", rest))
			}
			if err := openSection(path, false); err != nil {
				if err == errReentry {
					return nil, err
				}
				return nil, atLine(lineIdx-1, err)
			}
		} else {
			// key = value line — fast path for simple bare keys (no dots, no quotes).
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
				rest := strings.TrimLeft(trimmed[eqPos+1:], " \t")
				closeInlineTo(0)
				if err := markKey(key); err != nil {
					return nil, atLine(lineIdx-1, err)
				}
				if topNC() {
					buf.WriteByte(',')
				}
				writeJSONString(key, &buf)
				buf.WriteByte(':')
				consumed, err := writeTOMLValue(rest, lines, lineIdx-1, &buf)
				if err != nil {
					return nil, atLine(lineIdx-1, err)
				}
				lineIdx += consumed
				setTopNC(true)
			} else {
				// Slow path: dotted/quoted keys.
				path, rest, err := parseTOMLKeyPath(trimmed)
				if err != nil {
					return nil, atLine(lineIdx-1, err)
				}
				rest = strings.TrimSpace(rest)
				if !strings.HasPrefix(rest, "=") {
					return nil, atLine(lineIdx-1, fmt.Errorf("expected '=' after key, got: %s", rest))
				}
				rest = strings.TrimSpace(rest[1:])

				lastKey := path[len(path)-1]
				prefix := path[:len(path)-1]

				if len(prefix) > 0 {
					cd := 0
					for cd < len(prefix) && cd < len(inlineKeys) {
						if inlineKeys[cd] != prefix[cd] {
							break
						}
						cd++
					}
					closeInlineTo(cd)
					for i := cd; i < len(prefix); i++ {
						if err := markKey(prefix[i]); err != nil {
							return nil, atLine(lineIdx-1, err)
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
					return nil, atLine(lineIdx-1, err)
				}
				if topNC() {
					buf.WriteByte(',')
				}
				writeJSONString(lastKey, &buf)
				buf.WriteByte(':')
				consumed, err := writeTOMLValue(rest, lines, lineIdx-1, &buf)
				if err != nil {
					return nil, atLine(lineIdx-1, err)
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
