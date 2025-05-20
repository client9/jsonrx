package jsonrx

import "testing"

func TestString(t *testing.T) {
	cases := []testcase{
		{
			"abc",
			`"abc"`,
		},
		{
			"1\n2",
			`"1\n2"`,
		},
		{
			`1
2`,
			"\"1\\n2\"",
		},
		{
			`1	2`,
			"\"1\\t2\"",
		},
	}
	for _, tt := range cases {
		dst := []byte{}
		dst = appendRecodeString(dst, []byte(tt.in))
		/*
			if err != nil {
				t.Errorf("Got unexpected error: %v", err)
			}
		*/
		got := string(dst)
		if tt.out != got {
			t.Errorf("Expected %s got %s", tt.out, got)
		}
	}
}
