package tojson

import (
	"bytes"
	"testing"
)

func TestWriteNormalizedNumber(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		// leading + stripped
		{"+42", "42"},
		{"+1.5", "1.5"},
		// leading - preserved
		{"-7", "-7"},
		{"-1.5", "-1.5"},
		// leading dot → 0.
		{".5", "0.5"},
		{"+.5", "0.5"},
		{"-.5", "-0.5"},
		{".0", "0.0"},
		// trailing dot → .0
		{"5.", "5.0"},
		{"-5.", "-5.0"},
		{"0.", "0.0"},
		// trailing dot before exponent
		{"5.e4", "5.0e4"},
		{"5.E4", "5.0E4"},
		// leading zeros stripped
		{"01", "1"},
		{"01.5", "1.5"},
		{"00.5", "0.5"},
		// bare zero preserved
		{"0", "0"},
		{"+0", "0"},
		{"-0", "-0"},
		// scientific notation passes through
		{"1.5e4", "1.5e4"},
		{"1.5e+4", "1.5e+4"},
		{"1.5e-4", "1.5e-4"},
		{".5e4", "0.5e4"},
		// large values pass through without evaluation
		{"+99999999999999999999", "99999999999999999999"},
		{"1e309", "1e309"},
		{"+111111111111111111111123199999999999999999.0", "111111111111111111111123199999999999999999.0"},
	}
	for _, tt := range cases {
		var buf bytes.Buffer
		writeNormalizedNumber(&buf, []byte(tt.in))
		if got := buf.String(); got != tt.out {
			t.Errorf("writeNormalizedNumber(%q) = %q, want %q", tt.in, got, tt.out)
		}
	}
}
