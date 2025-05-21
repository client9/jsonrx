package jsonrx

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJsonNext(t *testing.T) {
	files := []string{}
	suf := []string{".txt"}

	whitelist := []string{

		/* positive and negative forms of 0x?? are likely never used */

		"negative-hexadecimal.json5",
		"negative-zero-hexadecimal.json5",
		"positive-hexadecimal.json5",
		"positive-zero-hexadecimal.json5",

		/* NaN is TBD */
		"nan.json5",

		/* This isn't JSON5 (or JSON) but allowed in JS */
		/* negative with leading zeros, e.g. -001       */
		/* Probably should strip off leading zeros      */
		"negative-noctal.js",

		/* MS-DOS \r issues */
		"escaped-crlf.json5",
		"escaped-cr.json5",
		"comment-cr.json5",
		"valid-whitespace.json5",
	}

	err := filepath.WalkDir("samples/json-next-tests", (func(path string, dir fs.DirEntry, err error) error {
		for _, s := range suf {
			if filepath.Ext(path) == s {
				files = append(files, path)
				return nil
			}
		}
		return nil
	}))
	if err != nil {
		t.Fatalf("filepath.WalkDir failed: %v", err)
	}

	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			for _, white := range whitelist {
				if white == filepath.Base(f) {
					t.Skip()
					return
				}
			}
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("%s: unable to read: %v", src, err)
			}
			idx := bytes.Index(src, []byte("---"))
			if idx == -1 {
				t.Fatalf("Couldn't find marker")
			}

			orig := src[:idx]
			want := src[idx+3:]

			data, err := Decode(orig)
			if err != nil {
				t.Fatalf("%s: Got unexpected error: %v", orig, err)
			}

			// now check that the output is valid JSON
			var out any
			err = json.Unmarshal(data, &out)
			if err != nil && err != io.EOF {
				t.Fatalf("Input of << %s >> decoded to << %s >> did not parse as JSON: %v", strings.TrimSpace(string(orig)), string(data), err)
			}

			// valid... does it match the expected?
			data2, err := Decode(want)
			if err != nil {
				t.Errorf("Unable to decode valid JSON %v", err)
			}
			data = bytes.TrimSpace(data)
			data2 = bytes.TrimSpace(data)
			if !bytes.Equal(data, data2) {
				t.Errorf("NO MATCH\n%s\n---%s\n", string(data), string(data2))
			}
		})
	}
}
