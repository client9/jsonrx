package jsonrx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
)

func BenchmarkDecodeTokens(b *testing.B) {
	data, err := os.ReadFile("samples/chromium/runtime_enabled_features.json5")
	if err != nil {
		log.Fatalf("Cant read file - %v", err)
	}

	for b.Loop() {
		rx := newTokenizer(data)
		for {
			_, err := rx.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("Unexpected error %v ", err)
			}
		}
	}
}
func BenchmarkDecodeFile(b *testing.B) {
	data, err := os.ReadFile("samples/chromium/runtime_enabled_features.json5")
	if err != nil {
		log.Fatalf("Cant read file - %v", err)
	}

	dst := bytes.Buffer{}
	dst.Grow(len(data))

	for b.Loop() {
		dst.Reset()
		err := DecodeAppend(&dst, data)
		if err != nil && err != io.EOF {
			b.Errorf("JsonRx - Decode failed %v", err)
		}
	}
}
func BenchmarkJson(b *testing.B) {
	data, err := os.ReadFile("samples/chromium/runtime_enabled_features.json5")
	if err != nil {
		log.Fatalf("Cant read file - %v", err)
	}
	var dst bytes.Buffer
	err = DecodeAppend(&dst, data)
	if err != nil && err != io.EOF {
		b.Errorf("JsonRx - Decode failed %v", err)
	}
	data = dst.Bytes()
	fmt.Printf("Final size: %d\n", len(data))
	for b.Loop() {
		var out map[string]any
		err := json.Unmarshal(data, &out)
		if err != nil && err != io.EOF {
			b.Fatalf("Decode failed %v", err)
		}
	}
}

func BenchmarkInt(b *testing.B) {
	data := []byte("123456789")
	out := &bytes.Buffer{}
	for b.Loop() {
		out.Reset()
		writeInt(out, data)
	}
}
func BenchmarkFloatFast(b *testing.B) {
	data := []byte("1.23456789")
	out := &bytes.Buffer{}
	for b.Loop() {
		out.Reset()
		writeFloat(out, data)
	}
}
func BenchmarkHex(b *testing.B) {
	data := []byte("0xDEADbeef")
	out := &bytes.Buffer{}
	for b.Loop() {
		out.Reset()
		writeHex(out, data)
	}
}
func BenchmarkString(b *testing.B) {
	data := []byte("\"a quoted string\"")
	out := &bytes.Buffer{}
	for b.Loop() {
		out.Reset()
		writeString(out, data)
	}
}
func BenchmarkQuotedFast(b *testing.B) {
	data := []byte("abcdefgh1234567890")
	out := &bytes.Buffer{}
	for b.Loop() {
		out.Reset()
		writeQuoted(out, data)
	}
}
func BenchmarkQuotedSlow(b *testing.B) {
	data := []byte("abcdefgh\\n1234567890")
	out := &bytes.Buffer{}
	for b.Loop() {
		out.Reset()
		writeQuoted(out, data)
	}
}
