package tojson

import "testing"

func requireParseError(t *testing.T, err error) *ParseError {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T: %v", err, err)
	}
	if pe.Line < 1 {
		t.Fatalf("expected line >= 1, got %d (msg: %s)", pe.Line, pe.Message)
	}
	if pe.Column < 1 {
		t.Fatalf("expected column >= 1, got %d (msg: %s)", pe.Column, pe.Message)
	}
	return pe
}
