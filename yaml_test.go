package tojson

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"testing"
)

//go:embed testdata/frontmatter1.yml
var frontmatter1YAML string

func BenchmarkFromYAML(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		raw, err := FromYAML([]byte(frontmatter1YAML))
		if err != nil {
			b.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			b.Fatal(err)
		}
	}
}

// roundtripYAML checks that FromYAML produces valid JSON matching wantJSON.
func roundtripYAML(t *testing.T, yaml, wantJSON string) {
	t.Helper()
	got, err := FromYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("FromYAML error: %v", err)
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
		t.Errorf("\ninput:  %s\ngot:    %s\nwant:   %s", yaml, gotNorm, wantNorm)
	}
}

func TestYAMLScalars(t *testing.T) {
	roundtripYAML(t, `hello`, `"hello"`)
	roundtripYAML(t, `"hello world"`, `"hello world"`)
	roundtripYAML(t, `'it''s fine'`, `"it's fine"`)
	roundtripYAML(t, `null`, `null`)
	roundtripYAML(t, `~`, `null`)
	roundtripYAML(t, `true`, `true`)
	roundtripYAML(t, `false`, `false`)
	roundtripYAML(t, `yes`, `true`)
	roundtripYAML(t, `no`, `false`)
	roundtripYAML(t, `42`, `42`)
	roundtripYAML(t, `3.14`, `3.14`)
	roundtripYAML(t, `-7`, `-7`)
	roundtripYAML(t, `1.5e10`, `1.5e10`)
}

func TestYAMLSimpleMapping(t *testing.T) {
	roundtripYAML(t, `
name: Alice
age: 30
active: true
`, `{"name":"Alice","age":30,"active":true}`)
}

func TestYAMLSimpleSequence(t *testing.T) {
	roundtripYAML(t, `
- rigid
- better for data interchange
`, `["rigid","better for data interchange"]`)
}

func TestYAMLMappingWithSequence(t *testing.T) {
	roundtripYAML(t, `
sample:
  - rigid
  - better for data interchange
`, `{"sample":["rigid","better for data interchange"]}`)
}

func TestYAMLCompactSequence(t *testing.T) {
	// Block sequence at the same indent as the parent mapping key (YAML compact notation).
	roundtripYAML(t, `
genres:
- mystery
- romance
tags:
- red
- blue
`, `{"genres":["mystery","romance"],"tags":["red","blue"]}`)

	// Mixed: some keys with indented sequences, some compact.
	roundtripYAML(t, `
a:
- 1
- 2
b:
  - 3
  - 4
`, `{"a":[1,2],"b":[3,4]}`)
}

func TestYAMLNestedMapping(t *testing.T) {
	roundtripYAML(t, `
person:
  name: Bob
  address:
    city: "New York"
    zip: "10001"
`, `{"person":{"name":"Bob","address":{"city":"New York","zip":"10001"}}}`)
}

func TestYAMLSequenceOfMappings(t *testing.T) {
	roundtripYAML(t, `
- name: Alice
  age: 30
- name: Bob
  age: 25
`, `[{"name":"Alice","age":30},{"name":"Bob","age":25}]`)
}

func TestYAMLComments(t *testing.T) {
	roundtripYAML(t, `
# top-level comment
name: Alice  # inline comment
age: 30
`, `{"name":"Alice","age":30}`)
}

func TestYAMLFrontmatter(t *testing.T) {
	roundtripYAML(t, `
---
title: Hello
tags:
  - go
  - yaml
`, `{"title":"Hello","tags":["go","yaml"]}`)
}

func TestYAMLQuotedKeys(t *testing.T) {
	roundtripYAML(t, `
"key with spaces": value
'another key': 42
`, `{"key with spaces":"value","another key":42}`)
}

func TestYAMLNullValues(t *testing.T) {
	roundtripYAML(t, `
a: null
b: ~
c:
`, `{"a":null,"b":null,"c":null}`)
}

func TestYAMLMixedNested(t *testing.T) {
	roundtripYAML(t, `
title: My Post
tags:
  - go
  - programming
meta:
  draft: false
  views: 0
`, `{"title":"My Post","tags":["go","programming"],"meta":{"draft":false,"views":0}}`)
}

func yamlJSONPassthrough(t *testing.T, input string) {
	t.Helper()
	got, err := FromYAML([]byte(input))
	if err != nil {
		t.Fatalf("FromYAML error: %v", err)
	}
	var gotV, wantV any
	if err := json.Unmarshal(got, &gotV); err != nil {
		t.Fatalf("output is not valid JSON %q: %v", got, err)
	}
	if err := json.Unmarshal([]byte(input), &wantV); err != nil {
		t.Fatalf("input is not valid JSON: %v", err)
	}
	gotNorm, _ := json.Marshal(gotV)
	wantNorm, _ := json.Marshal(wantV)
	if string(gotNorm) != string(wantNorm) {
		t.Errorf("\ninput: %s\ngot:   %s\nwant:  %s", input, gotNorm, wantNorm)
	}
}

func TestYAMLJSONPassthrough(t *testing.T) {
	yamlJSONPassthrough(t, `null`)
	yamlJSONPassthrough(t, `true`)
	yamlJSONPassthrough(t, `false`)
	yamlJSONPassthrough(t, `42`)
	yamlJSONPassthrough(t, `-7`)
	yamlJSONPassthrough(t, `3.14`)
	yamlJSONPassthrough(t, `1.5e10`)
	yamlJSONPassthrough(t, `"hello"`)
	yamlJSONPassthrough(t, `"line1\nline2\ttabbed"`)
	yamlJSONPassthrough(t, `"unicode \u0041"`)
	yamlJSONPassthrough(t, `{}`)
	yamlJSONPassthrough(t, `{"a":1}`)
	yamlJSONPassthrough(t, `{"a":1,"b":"hello","c":true,"d":null}`)
	yamlJSONPassthrough(t, `{"key with spaces":"value"}`)
	yamlJSONPassthrough(t, `[]`)
	yamlJSONPassthrough(t, `[1,2,3]`)
	yamlJSONPassthrough(t, `[true,false,null]`)
	yamlJSONPassthrough(t, `["a","b","c"]`)
	yamlJSONPassthrough(t, `{"a":{"b":{"c":42}}}`)
	yamlJSONPassthrough(t, `[[1,2],[3,4]]`)
	yamlJSONPassthrough(t, `{"nums":[1,2,3],"obj":{"x":true}}`)
	yamlJSONPassthrough(t, `[{"name":"Alice","age":30},{"name":"Bob","age":25}]`)
}

func TestYAMLFlowMapping(t *testing.T) {
	roundtripYAML(t, `{a: 1, b: hello}`, `{"a":1,"b":"hello"}`)
	roundtripYAML(t, `key: {x: true, y: null}`, `{"key":{"x":true,"y":null}}`)
	roundtripYAML(t, `{"key one": "val two"}`, `{"key one":"val two"}`)
	roundtripYAML(t, `{a: {b: {c: 42}}}`, `{"a":{"b":{"c":42}}}`)
	roundtripYAML(t, `{}`, `{}`)
	roundtripYAML(t, `
name: Alice
tags: {go: true, yaml: false}
`, `{"name":"Alice","tags":{"go":true,"yaml":false}}`)
	// single-quoted value with '' escape inside a flow mapping (exercises flowDepth i++ branch)
	roundtripYAML(t, `{key: 'it''s fine'}`, `{"key":"it's fine"}`)
}

func TestYAMLFlowSequence(t *testing.T) {
	roundtripYAML(t, `[1, 2, 3]`, `[1,2,3]`)
	roundtripYAML(t, `nums: [1, 2, 3]`, `{"nums":[1,2,3]}`)
	roundtripYAML(t, `[true, null, "hello", 3.14]`, `[true,null,"hello",3.14]`)
	roundtripYAML(t, `[[1, 2], [3, 4]]`, `[[1,2],[3,4]]`)
	roundtripYAML(t, `[]`, `[]`)
	roundtripYAML(t, `[{a: 1}, {b: 2}]`, `[{"a":1},{"b":2}]`)
	roundtripYAML(t, `
- [1, 2]
- [3, 4]
`, `[[1,2],[3,4]]`)
}

func TestYAMLFlowMultiLine(t *testing.T) {
	roundtripYAML(t, "key: {a: 1,\n      b: 2}", `{"key":{"a":1,"b":2}}`)
	roundtripYAML(t, "nums: [1,\n       2,\n       3]", `{"nums":[1,2,3]}`)
}

func TestYAMLParseError(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		line   int
		column int
	}{
		// mapping value: unterminated string
		{"mapping unterminated line 1", `key: "unterminated`, 1, 6},
		{"mapping unterminated line 3", "a: 1\nb: 2\nc: \"bad", 3, 4},
		// sequence item: unterminated string (after "- ")
		{"sequence unterminated", `- "bad`, 1, 3},
		// nested mapping value
		{"nested mapping unterminated", "foo:\n  bar: \"bad", 2, 8},
		// flow value: unterminated mapping
		{"flow unterminated mapping", "key: {a: 1", 1, 6},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := FromYAML([]byte(tc.input))
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

func TestYAMLEscapes(t *testing.T) {
	roundtripYAML(t, `msg: "line1\nline2\ttabbed"`, `{"msg":"line1\nline2\ttabbed"}`)
}

func TestYAMLLiteralBlockScalar(t *testing.T) {
	roundtripYAML(t, `
key: |
  line one
  line two
`, `{"key":"line one\nline two\n"}`)

	roundtripYAML(t, `
key: |-
  line one
  line two
`, `{"key":"line one\nline two"}`)

	roundtripYAML(t, "key: |+\n  line one\n  line two\n\n", `{"key":"line one\nline two\n\n"}`)

	roundtripYAML(t, `
a: |
  hello
  world
b: 42
`, `{"a":"hello\nworld\n","b":42}`)

	roundtripYAML(t, `
outer:
  inner: |
    indented
    content
`, `{"outer":{"inner":"indented\ncontent\n"}}`)

	roundtripYAML(t, `
- |
  first
- |
  second
`, `["first\n","second\n"]`)
}

func TestYAMLFoldedBlockScalar(t *testing.T) {
	roundtripYAML(t, `
key: >
  foo bar
  baz
`, `{"key":"foo bar baz\n"}`)

	roundtripYAML(t, `
key: >
  paragraph one

  paragraph two
`, `{"key":"paragraph one\nparagraph two\n"}`)

	roundtripYAML(t, `
key: >-
  foo
  bar
`, `{"key":"foo bar"}`)
}

func TestYAMLParseErrorString(t *testing.T) {
	e := &ParseError{Line: 3, Message: "bad token"}
	if got, want := e.Error(), "line 3: bad token"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestYAMLFlowSingleQuoted(t *testing.T) {
	roundtripYAML(t, `{key: 'hello world'}`, `{"key":"hello world"}`)
	roundtripYAML(t, `['a', 'b', 'c']`, `["a","b","c"]`)
	roundtripYAML(t, `{'my key': 42}`, `{"my key":42}`)
	roundtripYAML(t, `{'it''s': true}`, `{"it's":true}`)
}

func TestYAMLDoubleQuotedEscapes(t *testing.T) {
	roundtripYAML(t, `"\b\f"`, `"\u0008\u000c"`)
	roundtripYAML(t, `"\/"`, `"/"`)
	roundtripYAML(t, `"\q"`, `"\\q"`)
	roundtripYAML(t, `"\u41"`, `"\\u41"`)
	yamlJSONPassthrough(t, `"\uD800\uDC00"`)
}

func TestYAMLControlCharEncoding(t *testing.T) {
	got, err := FromYAML([]byte("v: \"\x01\""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `{"v":"\u0001"}` {
		t.Errorf("got %s, want {\"v\":\"\\u0001\"}", got)
	}
}

func TestYAMLIsNumberSigns(t *testing.T) {
	if !isYAMLNumber([]byte("+42")) {
		t.Error("isYAMLNumber(+42) should be true")
	}
	roundtripYAML(t, `1.5e+10`, `1.5e+10`)
	roundtripYAML(t, `2.0e-3`, `2.0e-3`)
}

func TestYAMLInlineMapSubValues(t *testing.T) {
	roundtripYAML(t, `
- name: Alice
  addr:
    city: NYC
`, `[{"name":"Alice","addr":{"city":"NYC"}}]`)

	roundtripYAML(t, `
- name: Alice
  tags: [go, yaml]
`, `[{"name":"Alice","tags":["go","yaml"]}]`)
}

func TestYAMLFlowErrors(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"unterminated mapping", `{key: value`},
		{"missing comma in mapping", `{a: {b: 1} {c: 2}}`},
		{"unterminated sequence", `[1, 2, 3`},
		{"missing comma in sequence", `[{a: 1} {b: 2}]`},
		{"unterminated string in flow", `{"key": "bad`},
		{"multiline unterminated mapping", "key: {a: 1,"},
		// unterminated double-quoted key — exercises parseFlowMapping flowParseKey error return
		{"unterminated quoted key in flow mapping", `{"bad key: 1}`},
		// unterminated double-quoted item — exercises parseFlowSequence flowParseItem error return
		{"unterminated quoted item in flow sequence", `["bad item]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := FromYAML([]byte(tc.input)); err == nil {
				t.Errorf("expected error for input %q", tc.input)
			}
		})
	}
}

func TestYAMLFlowDepthSingleQuoted(t *testing.T) {
	roundtripYAML(t,
		"key: ['it''s [ok]',\n      'fine']",
		`{"key":["it's [ok]","fine"]}`,
	)
	roundtripYAML(t, "key: {a: 1,\n\n     b: 2}", `{"key":{"a":1,"b":2}}`)
}

func TestYAMLConvertEmptyInput(t *testing.T) {
	for _, input := range []string{"", "  \n  \n", "# just a comment\n", "---\n"} {
		got, err := FromYAML([]byte(input))
		if err != nil || string(got) != "null" {
			t.Errorf("FromYAML(%q) = %s, %v; want null, nil", input, got, err)
		}
	}
}

func TestYAMLAtLine(t *testing.T) {
	if got := atLine(0, nil); got != nil {
		t.Errorf("atLine(nil) = %v, want nil", got)
	}
	pe := &ParseError{Line: 5, Message: "original"}
	if got := atLine(10, pe); got != pe {
		t.Errorf("atLine(ParseError) did not pass through: got %v", got)
	}
}

func TestYAMLDoubleQuotedEscapesMore(t *testing.T) {
	roundtripYAML(t, `"\r"`, `"\r"`)
	roundtripYAML(t, `"say \"hi\""`, `"say \"hi\""`)
	roundtripYAML(t, `"back\\slash"`, `"back\\slash"`)
	roundtripYAML(t, `"\u004a"`, `"J"`)
	roundtripYAML(t, `"\uGHIJ"`, `"\\uGHIJ"`)
}

func TestYAMLSingleQuotedValueWithDoubleQuote(t *testing.T) {
	roundtripYAML(t, `'say "hello"'`, `"say \"hello\""`)
}

func TestYAMLFlowTrailingComma(t *testing.T) {
	roundtripYAML(t, `{a: 1,}`, `{"a":1}`)
	roundtripYAML(t, `[1, 2,]`, `[1,2]`)
}

func TestYAMLBlockScalarTopLevel(t *testing.T) {
	roundtripYAML(t, "|\n  hello\n  world\n", `"hello\nworld\n"`)
}

func TestYAMLBlockScalarEmptyBody(t *testing.T) {
	roundtripYAML(t, "key: |\nnext: value", `{"key":"","next":"value"}`)
}

func TestYAMLSequenceEmptyDash(t *testing.T) {
	roundtripYAML(t, "-\n  name: Alice", `[{"name":"Alice"}]`)
	roundtripYAML(t, "-\n  nested: value\n-\n  nested: other", `[{"nested":"value"},{"nested":"other"}]`)
}

func TestYAMLQuotedMapKeys(t *testing.T) {
	roundtripYAML(t, `"key\n": value`, "{\"key\\n\":\"value\"}")
	roundtripYAML(t, `'it''s key': value`, `{"it's key":"value"}`)
}

func TestYAMLParseErrorPaths(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"top-level unterminated string", `"unterminated`},
		{"mapping value block error", "key:\n  \"unterminated"},
		{"sequence empty-dash block error", "-\n  \"unterminated"},
		{"sequence flow error", `- {key: "bad`},
		{"sequence inline map scalar error", `- name: "bad`},
		{"sequence inline map continuation error", "- name: Alice\n  age: \"bad"},
		{"inline map flow error", "- name: {key: \"bad"},
		{"inline map block error", "- name:\n  \"bad"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := FromYAML([]byte(tc.input)); err == nil {
				t.Errorf("expected error for input %q", tc.input)
			}
		})
	}
}

// TestParseFlowExprDirect exercises the empty-string and bare-scalar branches of
// parseFlowExpr, which are unreachable via isFlowValue but exist as defensive cases.
func TestParseFlowExprDirect(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "null"},
		{"   ", "null"},
		{"hello", `"hello"`},
		{"42", "42"},
	}
	for _, tc := range cases {
		var buf bytes.Buffer
		if err := parseFlowExpr([]byte(tc.in), &buf); err != nil {
			t.Errorf("parseFlowExpr(%q): unexpected error: %v", tc.in, err)
			continue
		}
		if got := buf.String(); got != tc.want {
			t.Errorf("parseFlowExpr(%q): got %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestFlowDepthSingleQuoteEscape exercises the i++ branch in flowDepth for ”
// (YAML single-quote escape). A bracket inside 'it”s}value' must not affect depth.
func TestFlowDepthSingleQuoteEscape(t *testing.T) {
	cases := []struct {
		s    string
		want int
	}{
		{`{key: 'it''s fine'}`, 0},
		// bracket inside a '' escape — without i++, the } would close the outer {
		{`{key: 'val''s}inner'}`, 0},
		// unbalanced to verify non-zero result still works
		{`{key: 'val''s fine'`, 1},
		// double-quote \" escape: without i++, the " after \ closes the string early
		// and the trailing } falls outside, miscounting depth
		{"{\"\\\"\"}", 0},
	}
	for _, tc := range cases {
		if got := flowDepth([]byte(tc.s)); got != tc.want {
			t.Errorf("flowDepth(%q): got %d, want %d", tc.s, got, tc.want)
		}
	}
}

func TestFromYAMLAppend(t *testing.T) {
	var buf []byte
	b := &bytes.Buffer{}
	b.WriteString(`[`)
	if err := FromYAMLAppend(b, []byte("42")); err != nil {
		t.Fatal(err)
	}
	b.WriteString(`]`)
	buf = b.Bytes()
	if string(buf) != `[42]` {
		t.Errorf("got %s, want [42]", buf)
	}
}
