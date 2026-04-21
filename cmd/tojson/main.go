// tojson converts JSON5/YAML/TOML from a file or stdin to JSON on stdout.
//
// Usage:
//
//	tojson file.yaml          # format inferred from extension
//	cat file.yaml | tojson -f yaml
//	tojson -pretty file.yaml  # pretty-printed JSON
//	tojson -compact file.yaml # explicit compact JSON
//	tojson -raw file.yaml     # raw output from conversion, no post-processing
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/client9/tojson"
)

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "tojson: "+format+"\n", args...)
	os.Exit(1)
}

func convert(format string, input []byte) ([]byte, error) {
	switch format {
	case "yaml", "yml":
		return tojson.FromYAML(input)
	case "toml":
		return tojson.FromTOML(input)
	case "json5", "json", "jsonc", "hjson":
		return tojson.FromJSON5(input)
	default:
		return nil, fmt.Errorf("unknown format %q", format)
	}
}

func main() {
	pretty := flag.Bool("pretty", false, "pretty-print JSON output")
	compact := flag.Bool("compact", false, "compact JSON output (default)")
	raw := flag.Bool("raw", false, "raw output from conversion, no post-processing")
	format := flag.String("f", "", "input format: yaml, toml, json5 (required when reading stdin)")
	flag.Parse()

	modeCount := 0
	for _, b := range []bool{*pretty, *compact, *raw} {
		if b {
			modeCount++
		}
	}
	if modeCount > 1 {
		fatalf("-pretty, -compact, and -raw are mutually exclusive")
	}

	var input []byte
	var err error
	var fmt_ string

	switch flag.NArg() {
	case 0:
		if *format == "" {
			fatalf("-f <format> is required when reading from stdin")
		}
		fmt_ = strings.ToLower(*format)
		input, err = io.ReadAll(os.Stdin)
		if err != nil {
			fatalf("reading stdin: %v", err)
		}
	case 1:
		filename := flag.Arg(0)
		input, err = os.ReadFile(filename)
		if err != nil {
			fatalf("%v", err)
		}
		if *format != "" {
			fmt_ = strings.ToLower(*format)
		} else {
			ext := strings.TrimPrefix(filepath.Ext(filename), ".")
			fmt_ = strings.ToLower(ext)
		}
	default:
		fatalf("usage: tojson [-pretty|-compact|-raw] [-f format] [file]")
	}

	out, err := convert(fmt_, input)
	if err != nil {
		fatalf("%v", err)
	}

	if *pretty {
		var v any
		if err := json.Unmarshal(out, &v); err != nil {
			fatalf("re-marshaling: %v", err)
		}
		out, err = json.MarshalIndent(v, "", "  ")
		if err != nil {
			fatalf("re-marshaling: %v", err)
		}
	}

	os.Stdout.Write(out)
	if !*raw {
		os.Stdout.WriteString("\n")
	}
}
