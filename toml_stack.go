package tojson

import (
	"bytes"
)

// streamFrame tracks one open TOML section on the streaming stack.
type streamFrame struct {
	key       []byte
	dotPath   string   // full dot-joined path (string for use as map key)
	isAoT     bool     // opened by [[...]]
	explicit  bool     // set when a [table] header explicitly named this frame
	needComma bool     // next entry in this object needs a leading comma
	usedKeys  [][]byte // lazily allocated; detects duplicate keys via bytes.Equal linear scan
}

// tomlLineClosedTables records every table path that has been popped off the
// section stack so far. The TOML spec forbids re-opening a table once another
// header has closed it (e.g. defining [a.b], then [a], then [a.b] again);
// openSection consults this set to detect such re-entry and reject the input.
type tomlLineClosedTables struct {
	root tomlLineClosedNode
}

// tomlLineClosedNode is one node in the trie of closed table paths. The root
// node carries no key; each child key is a single segment of a dotted header
// path. closed is true when the full path from the root to this node has been
// closed at least once.
type tomlLineClosedNode struct {
	key      []byte
	closed   bool
	children []tomlLineClosedNode
}

// find returns the child of n whose key equals the argument, or nil if no
// such child exists.
func (n *tomlLineClosedNode) find(key []byte) *tomlLineClosedNode {
	for i := range n.children {
		if bytes.Equal(n.children[i].key, key) {
			return &n.children[i]
		}
	}
	return nil
}

// child returns the child of n with the given key, creating and appending a
// fresh node when one does not already exist.
func (n *tomlLineClosedNode) child(key []byte) *tomlLineClosedNode {
	if child := n.find(key); child != nil {
		return child
	}
	n.children = append(n.children, tomlLineClosedNode{key: key})
	return &n.children[len(n.children)-1]
}

// mark records the path described by stack as closed. The root frame at
// stack[0] is skipped, so a stack of length one is a no-op (the document root
// is never closed). Intermediate ancestors are inserted into the trie without
// being marked closed; only the deepest node is flagged.
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

// contains reports whether the exact dotted path has previously been marked
// closed. It returns false if any segment of the path is missing from the
// trie, or if the terminal node exists but has not been flagged closed.
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

// reopens reports whether opening path would re-enter a table that has
// already been closed. commonDepth is the number of leading segments that
// path shares with the currently open section stack; only the suffix beyond
// that prefix is checked, since the shared prefix is by definition still
// open.
func (c *tomlLineClosedTables) reopens(path [][]byte, commonDepth int) bool {
	for i := commonDepth; i < len(path); i++ {
		if c.contains(path[:i+1]) {
			return true
		}
	}
	return false
}
