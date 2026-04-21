package tojson

import (
	"testing"
)

type testcase struct {
	in  string
	out string
}

// Valid JSON.. Decode should return return the input unchanged
func TestDecodeIdentity(t *testing.T) {
	cases := []string{
		"",
		"null",
		"true",
		"false",
		"123",
		"-123",
		"0.5",
		"-0.5",
		"\"abc\"",
		"\"abc\"",
		"{}",
		"[]",
		"[1,2,3]",
		"{\"foo\":\"bar\"}",
		"{\"foo\":\"bar\",\"rock\":\"roll\"}",
		"[{}]",
		"{\"foo\":\"bar\",\"rock\":[]}",
		"{\"foo\":\"bar\",\"rock\":{}}",
	}
	for _, tt := range cases {
		out, err := FromJSON5([]byte(tt))
		if err != nil {
			t.Errorf("Got unexpected error: %v", err)
		}
		got := string(out)
		if tt != got {
			t.Errorf("Expected %q got %q", tt, got)
		}
	}
}

// tests non-JSON with leading and trailing commas,
// and other degenerate forms
func TestDecodeComma(t *testing.T) {
	cases := []testcase{
		{
			"[1,2,3,]",
			"[1,2,3]",
		},
		{
			"[,1,2,3,]",
			"[1,2,3]",
		},
		{
			"{\"foo\":1,}",
			"{\"foo\":1}",
		},
		{
			"[,]", // degenerate case
			"[]",
		},
		{
			"[,1,]", // degenerate case
			"[1]",
		},
		{
			"{,}", // degenerate case
			"{}",
		},
	}
	for _, tt := range cases {
		out, err := FromJSON5([]byte(tt.in))
		if err != nil {
			t.Errorf("Got unexpected error: %v", err)
		}
		got := string(out)
		if tt.out != got {
			t.Errorf("Expected %q got %q", tt.out, got)
		}
	}
}
func TestDecodeComments(t *testing.T) {
	cases := []testcase{
		{
			`[1,2,
			// single line comment
			3,]`,
			"[1,2,3]",
		},
		{
			`[1,2,
			# single line comment
			3,]`,
			"[1,2,3]",
		},
		{
			`[1,2,
			/* multi 
			line
			comment
			*/
			3,]`,
			"[1,2,3]",
		},
	}
	for _, tt := range cases {
		out, err := FromJSON5([]byte(tt.in))
		if err != nil {
			t.Errorf("Got error: %v", err)
		}
		got := string(out)
		if tt.out != got {
			t.Errorf("Expected %q got %q", tt.out, got)
		}
	}
}
func TestDecodeNumbers(t *testing.T) {
	cases := []testcase{
		{
			"+123",
			"123",
		},
		{
			"+1.5",
			"1.5",
		},
		{
			"01",
			"1",
		},
		{
			"01.5",
			"1.5",
		},
		{
			".5",
			"0.5",
		},
		{
			"5.",
			"5",
		},
		{
			"0xFF",
			"255",
		},
		{
			"Infinity",
			"9007199254740991",
		},
		{
			"+Infinity",
			"9007199254740991",
		},
		{
			"-Infinity",
			"-9007199254740992",
		},
		// large integer → falls through to writeFloat and normalizes
		{"+99999999999999999", "1e+17"},
		// hex: full uint64 range
		{"0xFFFFFFFFFFFFFFFF", "18446744073709551615"},
	}

	for _, tt := range cases {
		out, err := FromJSON5([]byte(tt.in))
		if err != nil {
			t.Errorf("Got error: %v", err)
		}
		got := string(out)
		if tt.out != got {
			t.Errorf("Expected %q got %q", tt.out, got)
		}
	}
}

func TestDecodeNumberErrors(t *testing.T) {
	cases := []string{
		// hex overflow: 2^64, exceeds uint64
		"0x10000000000000000",
		// float overflow: +prefix strips JSON validity, value exceeds float64
		"+1e309",
		// float overflow in object value
		`{"x":+1e309}`,
		// float overflow in array
		`[+1e309]`,
		// hex overflow in object value
		`{"x":0x10000000000000000}`,
		// hex overflow in array
		`[0x10000000000000000]`,
	}
	for _, in := range cases {
		_, err := FromJSON5([]byte(in))
		if err == nil {
			t.Errorf("FromJSON5(%q): expected error, got nil", in)
		}
	}
}

func TestDecodeStrings(t *testing.T) {
	cases := []testcase{
		{
			"\"foo\"",
			"\"foo\"",
		},
		{
			"'foo'",
			"\"foo\"",
		},
		{
			"`foo`",
			"\"foo\"",
		},
		{
			"`foo\nbar`",
			"\"foo\\nbar\"",
		},
	}
	for _, tt := range cases {
		out, err := FromJSON5([]byte(tt.in))
		if err != nil {
			t.Errorf("Got error: %v", err)
		}
		got := string(out)
		if tt.out != got {
			t.Errorf("Expected %s got %s", tt.out, got)
		}
	}
}

func TestDecodeArrayValueTypes(t *testing.T) {
	cases := []testcase{
		{"[1.5]", "[1.5]"},
		{"[0xFF]", "[255]"},
		{"[[1,2]]", "[[1,2]]"},
		{`[{"x":1}]`, `[{"x":1}]`},
		{"[1.5,0xFF,[1],{}]", "[1.5,255,[1],{}]"},
	}
	for _, tt := range cases {
		out, err := FromJSON5([]byte(tt.in))
		if err != nil {
			t.Errorf("Decode(%q): unexpected error: %v", tt.in, err)
		}
		got := string(out)
		if tt.out != got {
			t.Errorf("Decode(%q): expected %q, got %q", tt.in, tt.out, got)
		}
	}
}

func TestDecodeImplicitComma(t *testing.T) {
	cases := []testcase{
		{`{"a":1 "b":2}`, `{"a":1,"b":2}`}, // adjacent object entries
		{"[1 2 3]", "[1,2,3]"},             // adjacent array values
		{"[[1][2]]", "[[1],[2]]"},          // adjacent containers in array
	}
	for _, tt := range cases {
		out, err := FromJSON5([]byte(tt.in))
		if err != nil {
			t.Errorf("Decode(%q): unexpected error: %v", tt.in, err)
		}
		got := string(out)
		if tt.out != got {
			t.Errorf("Decode(%q): expected %q, got %q", tt.in, tt.out, got)
		}
	}
}

func TestDecodeErrors(t *testing.T) {
	cases := []string{
		"[}",    // array closed with object brace
		"{]",    // object closed with array bracket
		`{"a"}`, // object key without colon
		"[:]",   // colon in array value position
	}
	for _, in := range cases {
		_, err := FromJSON5([]byte(in))
		if err == nil {
			t.Errorf("Decode(%q): expected error, got nil", in)
		}
	}
}

func TestDecodeNaN(t *testing.T) {
	cases := []string{
		"NaN",
		"+NaN",
		"-NaN",
		`{"x":NaN}`,
		`[1,NaN,2]`,
	}
	for _, in := range cases {
		_, err := FromJSON5([]byte(in))
		if err == nil {
			t.Errorf("Decode(%q): expected error for NaN, got nil", in)
		}
	}
}
