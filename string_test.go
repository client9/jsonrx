package tojson

import "testing"

func TestString(t *testing.T) {
	cases := []testcase{
		{"abc", `"abc"`},
		{"1\n2", `"1\n2"`},
		{"1\t2", `"1\t2"`},

		// \x?? hex escapes
		{"\\x41", `"A"`},      // printable ASCII
		{"\\x0a", `"\n"`},     // maps to named escape
		{"\\x00", `"\u0000"`}, // null byte
		{"\\xFF", `"\u00ff"`}, // high byte (uppercase hex)

		// escape sequence decoding
		{"\\n", `"\n"`},
		{"\\t", `"\t"`},
		{"\\b", `"\b"`},
		{"\\f", `"\f"`},
		{"\\r", `"\r"`},         // bare \r (not followed by \n)
		{"\\\\", `"\\"`},        // \\ → single backslash
		{"\\\"", `"\""`},        // \" → double quote
		{"\\a", `"\u0007"`},     // Go bell → \u0007
		{"\\v", `"\u000b"`},     // Go vertical tab → \u000b
		{"\\u0041", `"\u0041"`}, // \u pass-through

		// raw byte normalization
		{"\r\n", `"\n"`}, // CRLF → \n
		{"\r", `"\r"`},   // bare CR stays

		// multi-byte UTF-8 and safety escapes
		{"\u2028", `"\u2028"`}, // line separator
		{"\u2029", `"\u2029"`}, // paragraph separator
		{"\xff", `"\ufffd"`},   // invalid UTF-8 → replacement char
	}
	for _, tt := range cases {
		dst := appendRecodeString(nil, []byte(tt.in))
		got := string(dst)
		if tt.out != got {
			t.Errorf("appendRecodeString(%q): expected %s, got %s", tt.in, tt.out, got)
		}
	}
}

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
