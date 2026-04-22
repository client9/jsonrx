# tojson

Convert YAML, TOML, and JSON variants into standard JSON bytes, then unmarshal with Go's `encoding/json`.

[![Go Reference](https://pkg.go.dev/badge/github.com/client9/tojson.svg)](https://pkg.go.dev/github.com/client9/tojson)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/client9/tojson/actions/workflows/go.yml/badge.svg)](https://github.com/client9/tojson/actions)

## Why

- One library for the configuration and frontmatter formats you are most likely to encounter.
- Zero dependencies. `tojson` uses the Go standard library only.
- Convert everything to JSON bytes, then use the normal Go JSON ecosystem for unmarshaling, validation, and downstream tooling.
- No custom marshaling layer. Use `json` struct tags only.
- Standardized API and error handling across all supported formats.

Typical use case: accept a human-friendly config format, convert it to JSON, then reuse the normal Go JSON ecosystem for decoding and validation.

## Quick Start

Requires Go 1.24+.

```bash
go get github.com/client9/tojson
```

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/client9/tojson"
)

type Config struct {
	Name   string `json:"name"`
	Port   int    `json:"port"`
	Active bool   `json:"active"`
}

func main() {
	src := []byte("name: demo\nport: 8080\nactive: true\n")

	raw, err := tojson.FromYAML(src)
	if err != nil {
		log.Fatal(err)
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", cfg)
}
```

## Supported Inputs

### JSON variants

`FromJSONVariant` accepts standard JSON plus common JSON-derived extensions, including:

- comments (`//`, `/* */`, `#`)
- trailing commas
- unquoted object keys
- single-quoted and backtick strings
- hex numbers such as `0x2a`

This is intended for JSON5, JWCC, HuJSON, JSONC, and HanSON-style inputs that should normalize to strict JSON.

For more details, see [docs/json-variants.md](docs/json-variants.md).

### YAML

`FromYAML` supports a practical YAML subset aimed at config files and frontmatter.

Supported well:

- mappings with string keys
- sequences
- flow collections (`{}` and `[]`)
- quoted and plain scalars
- multi-line strings ('>' and '|')

Not supported:

- anchors and aliases
- tags
- complex keys (`? ...`)

If you need full YAML spec coverage or YAML AST manipulations, this package is the wrong tool.

### TOML

`FromTOML` accepts valid TOML documents and converts them to standard JSON bytes.

## API

```go
tojson.FromJSONVariant(src []byte) ([]byte, error)
tojson.FromYAML(src []byte) ([]byte, error)
tojson.FromTOML(src []byte) ([]byte, error)
```

All functions return compact JSON on success.

### Error Handling

Parse failures are returned as `*tojson.ParseError`, which includes a 1-based line number and a 1-based column number where the failure occurred.

```go
_, err := tojson.FromJSONVariant([]byte("{ unclosed: [1, 2, }"))
if err != nil {
	var pe *tojson.ParseError
	if errors.As(err, &pe) {
		log.Printf("parse error at line %d, col %d: %s", pe.Line, pe.Column, pe.Message)
	}
}
```

## Examples

### JSON variants

```go
src := []byte(`
{
  // comments are allowed
  unquoted: 'value',
  hex: 0x2a,
  trailing: [1, 2, 3,],
}
`)

raw, err := tojson.FromJSONVariant(src)
if err != nil {
	log.Fatal(err)
}

// raw == {"unquoted":"value","hex":42,"trailing":[1,2,3]}
```

### YAML

```go
type Config struct {
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}

src := []byte(`
title: Hello
tags:
  - go
  - yaml
`)

raw, err := tojson.FromYAML(src)
if err != nil {
	log.Fatal(err)
}

var cfg Config
if err := json.Unmarshal(raw, &cfg); err != nil {
	log.Fatal(err)
}
```

### TOML

```go
type Config struct {
	Title string `json:"title"`
	Port  int    `json:"port"`
}

src := []byte(`
title = "demo"
port = 8080
`)

raw, err := tojson.FromTOML(src)
if err != nil {
	log.Fatal(err)
}

var cfg Config
if err := json.Unmarshal(raw, &cfg); err != nil {
	log.Fatal(err)
}
```

## Performance

On frontmatter-style benchmark inputs in this repo, `FromYAML` used substantially less memory and was several times faster than common Go YAML packages. `FromTOML` used about half the memory of the TOML packages tested, with speed roughly comparable to `pelletier/go-toml` and faster than `BurntSushi/toml`.

See [docs/performance.md](docs/performance.md) for benchmark methodology, exact library comparisons, and raw numbers.

## CLI

The repo also includes a `tojson` command for testing and scripting:

```bash
go install github.com/client9/tojson/cmd/tojson@latest

tojson file.yaml
tojson file.toml
tojson file.json5
cat file.yaml | tojson -f yaml
tojson -pretty file.yaml
```

Use `-f` when reading from stdin so the input format is explicit.

## License

MIT. See [LICENSE.txt](LICENSE.txt)

