package jsonrx

import (
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJson5Json(t *testing.T) {
	files := []string{}
	suf := []string{".json", ".json5", ".js"}

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

	err := filepath.WalkDir("samples/json5-tests", (func(path string, dir fs.DirEntry, err error) error {
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
			data, err := Decode([]byte(src))
			if err != nil {
				t.Errorf("%s: Got unexpected error: %v", src, err)
			}

			// now check that the output is valid JSON
			var out any
			err = json.Unmarshal(data, &out)
			if err != nil && err != io.EOF {
				t.Errorf("Input of << %s >> decoded to << %s >> did not parse as JSON: %v", strings.TrimSpace(string(src)), string(data), err)
			}
		})
	}
}
