package tojson

import "testing"

func TestAppendRecodeString(t *testing.T) {
	cases := []testcase{
		{"abc", `"abc"`},
		{"1\n2", `"1\n2"`},
		{"1\t2", `"1\t2"`},
		{"\\x41", `"A"`},
		{"\\x0a", `"\n"`},
		{"\\x00", `"\u0000"`},
		{"\\xFF", `"\u00ff"`},
		{"\\n", `"\n"`},
		{"\\t", `"\t"`},
		{"\\b", `"\b"`},
		{"\\f", `"\f"`},
		{"\\r", `"\r"`},
		{"\\\\", `"\\"`},
		{"\\\"", `"\""`},
		{"\\a", `"\u0007"`},
		{"\\v", `"\u000b"`},
		{"\\u0041", `"\u0041"`},
		{"\r\n", `"\n"`},
		{"\r", `"\r"`},
		{"\u2028", `"\u2028"`},
		{"\u2029", `"\u2029"`},
		{"\xff", `"\ufffd"`},
	}
	for _, tt := range cases {
		dst := appendRecodeString(nil, []byte(tt.in))
		got := string(dst)
		if tt.out != got {
			t.Errorf("appendRecodeString(%q): expected %s, got %s", tt.in, tt.out, got)
		}
	}
}
