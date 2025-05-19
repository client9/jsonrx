package main

import (
	"bytes"
	"io"
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
		var dst bytes.Buffer
		Decode(&dst, []byte(tt))
		got := dst.String()
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
		var dst bytes.Buffer
		err := Decode(&dst, []byte(tt.in))
		if err != nil && err != io.EOF {
			t.Errorf("Got error: %v", err)
		}
		got := dst.String()
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
			/* multi 
			   line
			   comment
			*/
			3,]`,
			"[1,2,3]",
		},
	}
	for _, tt := range cases {
		var dst bytes.Buffer
		err := Decode(&dst, []byte(tt.in))

		if err != nil && err != io.EOF {
			t.Errorf("Got error: %v", err)
		}
		got := dst.String()
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
			"0xFF",
			"255",
		},
	}

	for _, tt := range cases {
		var dst bytes.Buffer
		err := Decode(&dst, []byte(tt.in))

		if err != nil && err != io.EOF {
			t.Errorf("Got error: %v", err)
		}
		got := dst.String()
		if tt.out != got {
			t.Errorf("Expected %q got %q", tt.out, got)
		}
	}
}
