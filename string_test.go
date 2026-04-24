package tojson

import "testing"

func TestAppendString(t *testing.T) {
	cases := []testcase{
		{"abc", `"abc"`},
		{"\x08", `"\b"`},       // backspace
		{"\x0c", `"\f"`},       // form feed
		{"\x09", `"\t"`},       // tab
		{"\x01", `"\u0001"`},   // control char → \u00xx
		{"a\r\nb", `"a\nb"`},   // CRLF → \n
		{"\r", `"\r"`},         // bare CR
		{"\u2028", `"\u2028"`}, // line separator
		{"\u2029", `"\u2029"`}, // paragraph separator
		{"\xff", `"\ufffd"`},   // invalid UTF-8 → replacement char
		{"café", `"café"`},     // valid multi-byte passthrough
	}
	for _, tt := range cases {
		dst := appendString(nil, []byte(tt.in))
		got := string(dst)
		if tt.out != got {
			t.Errorf("appendString(%q): expected %s, got %s", tt.in, tt.out, got)
		}
	}
}
