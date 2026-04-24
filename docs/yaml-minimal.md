# Minimal YAML

`tojson.FromYAML` accepts a well-defined YAML subset. This document is the spec: if something is listed here, it works; if it isn't listed, it isn't supported.

```go
raw, err := tojson.FromYAML(yamlInput)  // → compact JSON bytes
if err != nil { ... }

var cfg MyConfig
json.Unmarshal(raw, &cfg)               // standard JSON unmarshalling
```

Because the output is plain JSON, you only need `json` struct tags — no separate `yaml` tags, and the full JSON ecosystem (validation, pretty-printing, streaming) works as-is.

## Structure

- Block mappings with string keys (`key: value`)
- Block sequences (`- item`)
- Flow mappings (`{key: value, ...}`)
- Flow sequences (`[a, b, c]`)
- Arbitrary nesting of the above

## Scalars

**Null**: the bare words `null`, `Null`, `NULL`, or an empty value.

**Booleans**: `true`, `True`, `TRUE`, `false`, `False`, `FALSE`.

**Integers**: `[-+]?(0|[1-9][0-9]*)` — decimal only, no leading zeros except bare `0`.
Leading `+` is accepted and stripped on output. `0012` is a string, not a number.

**Floats**: `[-+]?[0-9]*\.[0-9]*([eE][-+]?[0-9]+)?` — decimal only.
Leading `+` stripped on output. Normalized forms: `.5` → `0.5`, `5.` → `5.0`, `5.e4` → `5.0e4`.
Large values pass through without evaluation — `1e309` stays `1e309`, not `Infinity`.

**Strings**

- *Unquoted*: any value not recognized as null, boolean, or number is a string.
- *Single-quoted* (`'...'`): content is literal; `''` is the only escape (a literal single quote).
- *Double-quoted* (`"..."`): Go string literal rules via `strconv.Unquote`. Supported escapes: `\n \t \r \\ \" \a \b \f \v \uNNNN \UNNNNNNNN \xNN`. YAML-specific escapes (`\/ \e \N \L \P`) and surrogate pairs (`𐀀`) are not supported and produce an error.
- *Block scalars*: literal (`|`) preserves newlines; folded (`>`) folds newlines to spaces.

## Comments

`#` line comments, when preceded by whitespace.

## Internally configurable

Controlled by constants in `yaml_scalar.go`:

- [x] Tabs in indentation, counted as N spaces (`yamlTabWidth`, default 2; set to -1 to forbid)
- [x] YAML 1.1 boolean aliases: `yes`/`no`/`on`/`off` → `true`/`false` (`yamlBoolAliases`, default on)
- [ ] `~` as null (`yamlTildeNull`, default off)

## Out of scope

Anchors and aliases (`&name` / `*name`) are the most commonly encountered YAML feature outside this spec — they are not supported.

Everything else in the YAML specification not listed above — tags, complex keys, octal and hex integers, sexagesimal numbers, timestamps, multi-document streams — is also out of scope.

## Alternatives

### [github.com/goccy/go-yaml](https://github.com/goccy/go-yaml)

Convert the YAML file into an AST and then 

### [github.com/go-yaml/v3](https://github.com/go-yaml/yaml)

This is the standard YAML library for Go.   It was went into a "unmaintained state" in 2025.

### [github.com/yaml/go-yaml](https://github.com/yaml/go-yaml)

Note it's `yaml/go-yaml`, while the previous is `go-yaml/yaml`

This is the new maintained, official version of YAML parsing.  v4 has not been released as of April 2026.

### [sigs.k8s.io/yaml](https://github.com/kubernetes-sigs/yaml)

Part of the Kubernetes Project, it wraps `yaml/go-yaml`.

- Unmarshal input using `yaml/go-yaml` into an `any`.
- It does some conversion, yaml keys can be anything, while in JSON they must be strings.
- Then uses encoding/json to marshal that into bytes
- The uses JSON unmarshalling any it into `structs`. 

The benefit being only json struct tags are needed, and can leveage all the JSON unmarhsalling infrastructure.  That makes sense, but this implementation is 7x slower than `tojson`.

The concept is described [here](https://web.archive.org/web/20190603050330/http://ghodss.com/2014/the-right-way-to-handle-yaml-in-golang/).
