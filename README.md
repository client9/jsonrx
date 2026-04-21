# tojson

Convert YAML, TOML, and JSON variants to standard JSON bytes — no dependencies, no reflection, one decode path.

[![Go Reference](https://pkg.go.dev/badge/github.com/client9/tojson.svg)](https://pkg.go.dev/github.com/client9/tojson)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/client9/tojson/actions/workflows/go.yml/badge.svg)](https://github.com/client9/tojson/actions)

## Why

- **Zero dependencies** — stdlib only, nothing added to your module graph
- **One decode path** — all formats produce JSON bytes; unmarshal with `encoding/json` as usual
- **Only `json` struct tags** — no separate `yaml` or `toml` tags needed
- **Inspectable output** — the JSON bytes can be logged, stored, or passed to any JSON tool

## Install

Requires Go 1.24+.

```bash
go get github.com/client9/tojson
```

## How It Works

Every `From*` function converts the source bytes to standard JSON, then you unmarshal normally:

```go
raw, err := tojson.FromYAML(src)   // → compact JSON bytes
json.Unmarshal(raw, &cfg)          // standard encoding/json
```

## API

```go
tojson.FromJSONVariant(src []byte) ([]byte, error)
tojson.FromYAML(src []byte) ([]byte, error)
tojson.FromTOML(src []byte) ([]byte, error)
```

## Usage

### Basic example

```go
type Config struct {
	Name   string `json:"name"`
	Port   int    `json:"port"`
	Active bool   `json:"active"`
}

src := []byte("name: demo\nport: 8080\nactive: true\n")

raw, err := tojson.FromYAML(src)
// raw → {"name":"demo","port":8080,"active":true}

var cfg Config
json.Unmarshal(raw, &cfg)
// cfg → {Name:demo Port:8080 Active:true}
```

### JSON variants

`FromJSONVariant` accepts standard JSON plus JSON5, JWCC, HuJSON, JSONC, and HanSON — any input with comments, unquoted keys, trailing commas, hex literals, or single-quoted strings.

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
// raw → {"unquoted":"value","hex":42,"trailing":[1,2,3]}
```

### YAML

> **Subset only.** Anchors, aliases, tags, and complex keys (`? ...`) are not supported.

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
// raw → {"title":"Hello","tags":["go","yaml"]}

var cfg Config
json.Unmarshal(raw, &cfg)
```

### TOML

Parses all TOML files.

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
// raw → {"title":"demo","port":8080}

var cfg Config
json.Unmarshal(raw, &cfg)
```

## Error handling

All functions return `*tojson.ParseError` on failure, with a 1-based line number and, when available, a 1-based column number. When the column is not available, `ParseError.Column` is `0`.

```go
_, err := tojson.FromJSONVariant([]byte("{ unclosed: [1, 2, }"))
if err != nil {
	var pe *tojson.ParseError
	if errors.As(err, &pe) {
		log.Printf("parse error at line %d, col %d: %s", pe.Line, pe.Column, pe.Message)
	}
}
```

## CLI

A `tojson` command is included for testing and scripting:

```bash
go install github.com/client9/tojson/cmd/tojson@latest

tojson file.yaml          # infer format from extension
tojson file.toml
tojson file.json5
cat file.yaml | tojson -f yaml    # explicit format for stdin
tojson -pretty file.yaml          # pretty-printed output
```

## License

MIT
