package tojson

import (
	"bytes"
	"encoding/json"
	"testing"
)

func roundtripTOML(t *testing.T, toml, wantJSON string) {
	t.Helper()
	got, err := FromTOML([]byte(toml))
	if err != nil {
		t.Fatalf("FromTOML error: %v", err)
	}
	var v any
	if err := json.Unmarshal(got, &v); err != nil {
		t.Fatalf("invalid JSON %q: %v", got, err)
	}
	gotNorm, _ := json.Marshal(v)
	var wantV any
	if err := json.Unmarshal([]byte(wantJSON), &wantV); err != nil {
		t.Fatalf("bad wantJSON %q: %v", wantJSON, err)
	}
	wantNorm, _ := json.Marshal(wantV)
	if string(gotNorm) != string(wantNorm) {
		t.Errorf("\ninput:  %s\ngot:    %s\nwant:   %s", toml, gotNorm, wantNorm)
	}
}

// --------------------------------------------------------------------------
// Scalars
// --------------------------------------------------------------------------

func TestTOMLBasicStrings(t *testing.T) {
	roundtripTOML(t, `key = "hello"`, `{"key":"hello"}`)
	roundtripTOML(t, `key = "tab\there"`, `{"key":"tab\there"}`)
	roundtripTOML(t, `key = "newline\nhere"`, `{"key":"newline\nhere"}`)
	roundtripTOML(t, `key = "backslash\\"`, `{"key":"backslash\\"}`)
	roundtripTOML(t, `key = "quote\""`, `{"key":"quote\""}`)
	roundtripTOML(t, `key = "\u0041"`, `{"key":"A"}`)
	roundtripTOML(t, `key = "\U00000041"`, `{"key":"A"}`)
}

func TestTOMLLiteralStrings(t *testing.T) {
	roundtripTOML(t, `key = 'hello world'`, `{"key":"hello world"}`)
	roundtripTOML(t, `key = 'no \n escape'`, `{"key":"no \\n escape"}`)
	roundtripTOML(t, `key = 'C:\Users\tom'`, `{"key":"C:\\Users\\tom"}`)
}

func TestTOMLMultilineBasic(t *testing.T) {
	roundtripTOML(t, "key = \"\"\"\nline one\nline two\n\"\"\"", `{"key":"line one\nline two\n"}`)
	// line-ending backslash trims whitespace
	roundtripTOML(t, "key = \"\"\"hello \\\n   world\"\"\"", `{"key":"hello world"}`)
}

func TestTOMLMultilineLiteral(t *testing.T) {
	roundtripTOML(t, "key = '''\nline one\nline two\n'''", `{"key":"line one\nline two\n"}`)
	// no escape processing
	roundtripTOML(t, "key = '''no \\n escape'''", `{"key":"no \\n escape"}`)
}

func TestTOMLIntegers(t *testing.T) {
	roundtripTOML(t, `n = 42`, `{"n":42}`)
	roundtripTOML(t, `n = -7`, `{"n":-7}`)
	roundtripTOML(t, `n = +99`, `{"n":99}`)
	roundtripTOML(t, `n = 0`, `{"n":0}`)
	roundtripTOML(t, `n = 1_000_000`, `{"n":1000000}`)
	roundtripTOML(t, `n = 0xFF`, `{"n":255}`)
	roundtripTOML(t, `n = 0o17`, `{"n":15}`)
	roundtripTOML(t, `n = 0b1010`, `{"n":10}`)
	roundtripTOML(t, `n = 0xDEAD_BEEF`, `{"n":3735928559}`)
}

func TestTOMLFloats(t *testing.T) {
	roundtripTOML(t, `f = 3.14`, `{"f":3.14}`)
	roundtripTOML(t, `f = -0.001`, `{"f":-0.001}`)
	roundtripTOML(t, `f = 5e22`, `{"f":5e22}`)
	roundtripTOML(t, `f = 1e+99`, `{"f":1e+99}`)
	roundtripTOML(t, `f = 6.626e-34`, `{"f":6.626e-34}`)
	roundtripTOML(t, `f = 1_0.0`, `{"f":10.0}`)
}

func TestTOMLBooleans(t *testing.T) {
	roundtripTOML(t, `a = true`, `{"a":true}`)
	roundtripTOML(t, `a = false`, `{"a":false}`)
}

func TestTOMLDatetimes(t *testing.T) {
	roundtripTOML(t, `dt = 1979-05-27T07:32:00Z`, `{"dt":"1979-05-27T07:32:00Z"}`)
	roundtripTOML(t, `d = 1979-05-27`, `{"d":"1979-05-27"}`)
	roundtripTOML(t, `t = 07:32:00`, `{"t":"07:32:00"}`)
	roundtripTOML(t, `dt = 1979-05-27T07:32:00.999999-08:00`, `{"dt":"1979-05-27T07:32:00.999999-08:00"}`)
}

// --------------------------------------------------------------------------
// Keys
// --------------------------------------------------------------------------

func TestTOMLKeyForms(t *testing.T) {
	roundtripTOML(t, `bare_key = 1`, `{"bare_key":1}`)
	roundtripTOML(t, `bare-key = 1`, `{"bare-key":1}`)
	roundtripTOML(t, `"quoted key" = 1`, `{"quoted key":1}`)
	roundtripTOML(t, `'literal key' = 1`, `{"literal key":1}`)
}

func TestTOMLDottedKeys(t *testing.T) {
	roundtripTOML(t, `a.b = 1`, `{"a":{"b":1}}`)
	roundtripTOML(t, `a.b.c = 1`, `{"a":{"b":{"c":1}}}`)
	roundtripTOML(t, "a.b = 1\na.c = 2", `{"a":{"b":1,"c":2}}`)
}

// --------------------------------------------------------------------------
// Standard tables
// --------------------------------------------------------------------------

func TestTOMLSimpleTable(t *testing.T) {
	roundtripTOML(t, "[server]\nhost = \"localhost\"\nport = 8080",
		`{"server":{"host":"localhost","port":8080}}`)
}

func TestTOMLDottedTableHeader(t *testing.T) {
	roundtripTOML(t, "[a.b.c]\nkey = 1", `{"a":{"b":{"c":{"key":1}}}}`)
}

func TestTOMLMultipleTables(t *testing.T) {
	roundtripTOML(t,
		"[a]\nx = 1\n[b]\ny = 2",
		`{"a":{"x":1},"b":{"y":2}}`)
}

func TestTOMLImplicitTables(t *testing.T) {
	// [a.b] creates 'a' implicitly; then [a] can add sibling keys
	roundtripTOML(t,
		"[a.b]\nx = 1\n[a]\ny = 2",
		`{"a":{"b":{"x":1},"y":2}}`)
}

func TestTOMLTableReentry(t *testing.T) {
	// Critical: [a] ... [b] ... [a.c] must re-enter the 'a' object
	roundtripTOML(t,
		"[a]\nx = 1\n[b]\ny = 2\n[a.c]\nz = 3",
		`{"a":{"x":1,"c":{"z":3}},"b":{"y":2}}`)
}

// --------------------------------------------------------------------------
// Array of tables
// --------------------------------------------------------------------------

func TestTOMLArrayOfTables(t *testing.T) {
	roundtripTOML(t,
		"[[products]]\nname = \"Hammer\"\n[[products]]\nname = \"Nail\"",
		`{"products":[{"name":"Hammer"},{"name":"Nail"}]}`)
}

func TestTOMLNestedAoT(t *testing.T) {
	roundtripTOML(t,
		"[[fruits]]\nname = \"apple\"\n[[fruits.varieties]]\nname = \"red\"\n[[fruits.varieties]]\nname = \"green\"",
		`{"fruits":[{"name":"apple","varieties":[{"name":"red"},{"name":"green"}]}]}`)
}

func TestTOMLMixedTableAndAoT(t *testing.T) {
	roundtripTOML(t,
		"[a]\nx = 1\n[[a.items]]\nname = \"A\"\n[[a.items]]\nname = \"B\"",
		`{"a":{"x":1,"items":[{"name":"A"},{"name":"B"}]}}`)
}

// --------------------------------------------------------------------------
// Inline tables
// --------------------------------------------------------------------------

func TestTOMLInlineTable(t *testing.T) {
	roundtripTOML(t, `point = {x = 1, y = 2}`, `{"point":{"x":1,"y":2}}`)
	roundtripTOML(t, `empty = {}`, `{"empty":{}}`)
	roundtripTOML(t, `nested = {a = {b = 42}}`, `{"nested":{"a":{"b":42}}}`)
}

func TestTOMLInlineTableDottedKeys(t *testing.T) {
	roundtripTOML(t, `t = {a.b = 1, a.c = 2}`, `{"t":{"a":{"b":1,"c":2}}}`)
}

// --------------------------------------------------------------------------
// Inline arrays
// --------------------------------------------------------------------------

func TestTOMLInlineArray(t *testing.T) {
	roundtripTOML(t, `nums = [1, 2, 3]`, `{"nums":[1,2,3]}`)
	roundtripTOML(t, `mixed = [1, "two", true]`, `{"mixed":[1,"two",true]}`)
	roundtripTOML(t, `nested = [[1, 2], [3, 4]]`, `{"nested":[[1,2],[3,4]]}`)
	roundtripTOML(t, `empty = []`, `{"empty":[]}`)
	roundtripTOML(t, `trailing = [1, 2,]`, `{"trailing":[1,2]}`)
}

func TestTOMLInlineArrayMultiline(t *testing.T) {
	roundtripTOML(t, "nums = [\n  1,\n  2,\n  3,\n]", `{"nums":[1,2,3]}`)
}

// --------------------------------------------------------------------------
// Comments
// --------------------------------------------------------------------------

func TestTOMLComments(t *testing.T) {
	roundtripTOML(t, "# full line comment\nkey = 1", `{"key":1}`)
	roundtripTOML(t, `key = 1 # inline comment`, `{"key":1}`)
	roundtripTOML(t, "# comment\n[section]\n# another\nkey = 2", `{"section":{"key":2}}`)
}

// --------------------------------------------------------------------------
// Empty input
// --------------------------------------------------------------------------

func TestTOMLEmptyInput(t *testing.T) {
	for _, input := range []string{"", "  \n  ", "# just a comment"} {
		got, err := FromTOML([]byte(input))
		if err != nil {
			t.Errorf("FromTOML(%q) error: %v", input, err)
			continue
		}
		if string(got) != "{}" {
			t.Errorf("FromTOML(%q) = %s, want {}", input, got)
		}
	}
}

// --------------------------------------------------------------------------
// Error cases
// --------------------------------------------------------------------------

func TestTOMLErrorDuplicateKey(t *testing.T) {
	if _, err := FromTOML([]byte("a = 1\na = 2")); err == nil {
		t.Error("expected error for duplicate key")
	}
}

func TestTOMLErrorDuplicateTable(t *testing.T) {
	if _, err := FromTOML([]byte("[a]\n[a]")); err == nil {
		t.Error("expected error for duplicate table")
	}
}

func TestTOMLErrorScalarAsTable(t *testing.T) {
	if _, err := FromTOML([]byte("a = 1\n[a]")); err == nil {
		t.Error("expected error for scalar redefined as table")
	}
}

func TestTOMLErrorInf(t *testing.T) {
	if _, err := FromTOML([]byte("f = inf")); err == nil {
		t.Error("expected error for inf")
	}
	if _, err := FromTOML([]byte("f = -inf")); err == nil {
		t.Error("expected error for -inf")
	}
}

func TestTOMLErrorNaN(t *testing.T) {
	if _, err := FromTOML([]byte("f = nan")); err == nil {
		t.Error("expected error for nan")
	}
}

func TestTOMLErrorLeadingZero(t *testing.T) {
	if _, err := FromTOML([]byte("n = 01")); err == nil {
		t.Error("expected error for leading zero integer")
	}
}

func TestTOMLErrorUnterminatedString(t *testing.T) {
	if _, err := FromTOML([]byte(`key = "unclosed`)); err == nil {
		t.Error("expected error for unterminated string")
	}
}

func TestTOMLErrorInvalidEscape(t *testing.T) {
	if _, err := FromTOML([]byte(`key = "\q"`)); err == nil {
		t.Error("expected error for invalid escape")
	}
}

func TestTOMLErrorInlineTableTrailingComma(t *testing.T) {
	if _, err := FromTOML([]byte(`t = {a = 1,}`)); err == nil {
		t.Error("expected error for trailing comma in inline table")
	}
}

// --------------------------------------------------------------------------
// ParseError line numbers
// --------------------------------------------------------------------------

func TestTOMLParseErrorLineNumber(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		line   int
		column int
	}{
		// simple value error: streaming fast path
		{"unterminated string line 2", "a = 1\nb = \"unclosed", 2, 5},
		// value error first line
		{"inf not valid", "b = inf", 1, 5},
		// malformed table header: structural, column at start of content
		{"malformed table header", "[bad header", 1, 1},
		// dotted key value error: streaming slow path
		{"dotted key unterminated", `a.b = "unclosed`, 1, 7},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := FromTOML([]byte(tc.input))
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			pe, ok := err.(*ParseError)
			if !ok {
				t.Fatalf("expected *ParseError, got %T: %v", err, err)
			}
			if pe.Line != tc.line {
				t.Errorf("expected line %d, got %d (msg: %s)", tc.line, pe.Line, pe.Message)
			}
			if pe.Column != tc.column {
				t.Errorf("expected column %d, got %d (msg: %s)", tc.column, pe.Column, pe.Message)
			}
		})
	}
}

func TestTOMLParseErrorString(t *testing.T) {
	// Column=0 means not available: omit from message.
	e := &ParseError{Line: 5, Message: "bad token"}
	if got, want := e.Error(), "line 5: bad token"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// Column>0: include in message.
	e2 := &ParseError{Line: 5, Column: 3, Message: "bad token"}
	if got, want := e2.Error(), "line 5, column 3: bad token"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --------------------------------------------------------------------------
// FromTOMLAppend
// --------------------------------------------------------------------------

func TestFromTOMLAppend(t *testing.T) {
	var b bytes.Buffer
	b.WriteString(`[`)
	if err := FromTOMLAppend(&b, []byte("n = 42")); err != nil {
		t.Fatal(err)
	}
	b.WriteString(`]`)
	if got := b.String(); got != `[{"n":42}]` {
		t.Errorf("got %s, want [{\"n\":42}]", got)
	}
}
