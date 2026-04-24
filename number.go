package tojson

import "bytes"

// writeNormalizedNumber writes b to out as a valid JSON number.
// It strips a leading +, strips leading zeros from the integer part,
// and normalizes leading/trailing dots (.5→0.5, 5.→5.0, 5.e4→5.0e4).
// No numeric evaluation is performed — large integers and high-exponent
// floats pass through exactly as written.
func writeNormalizedNumber(out *bytes.Buffer, b []byte) {
	// strip leading +
	if len(b) > 0 && b[0] == '+' {
		b = b[1:]
	}

	// split off sign
	var sign []byte
	if len(b) > 0 && b[0] == '-' {
		sign, b = b[:1], b[1:]
	}

	// find end of integer part
	intEnd := 0
	for intEnd < len(b) && b[intEnd] >= '0' && b[intEnd] <= '9' {
		intEnd++
	}
	intPart, rest := b[:intEnd], b[intEnd:]

	out.Write(sign)

	if len(intPart) == 0 {
		// leading dot: .5 → 0.5
		out.WriteByte('0')
	} else {
		// strip leading zeros from integer part (keep at least one digit)
		for len(intPart) > 1 && intPart[0] == '0' {
			intPart = intPart[1:]
		}
		out.Write(intPart)
	}

	if len(rest) == 0 {
		return
	}

	if rest[0] != '.' {
		// exponent with no decimal point
		out.Write(rest)
		return
	}

	// trailing dot (5. or 5.e4) → insert 0 after dot
	if len(rest) == 1 || rest[1] == 'e' || rest[1] == 'E' {
		out.WriteByte('.')
		out.WriteByte('0')
		out.Write(rest[1:])
		return
	}

	out.Write(rest)
}
