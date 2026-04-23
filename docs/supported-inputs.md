# Supported Inputs

## JSON variants

`FromJSONVariant` accepts standard JSON plus common JSON-derived extensions, including:

- comments (`//`, `/* */`, `#`)
- trailing commas
- unquoted object keys
- single-quoted and backtick strings
- hex numbers such as `0x2a`

This is intended for JSON5, JWCC, HuJSON, JSONC, and HanSON-style inputs that should normalize to strict JSON.

For more details, see [json-variants.md](json-variants.md).

## YAML

`FromYAML` supports a practical YAML subset aimed at config files and front matter.

Supported well:

- mappings with string keys
- sequences
- flow collections (`{}` and `[]`)
- quoted and plain scalars
- multi-line strings (`>` and `|`)

Not supported:

- anchors and aliases
- tags
- complex keys (`? ...`)

If you need full YAML spec coverage or YAML AST manipulations, this package is the wrong tool.

## TOML

`FromTOML` accepts valid TOML documents and converts them to standard JSON bytes.

## Front matter

`FromFrontMatter` handles documents that embed metadata before the main content, as used by Hugo, Jekyll, and similar static site generators. It detects the format from the opening sentinel line, converts the metadata block to JSON, and returns the metadata and body separately.

Supported sentinel pairs:

| Opening    | Closing | Format |
|------------|---------|--------|
| `---`      | `---`   | YAML   |
| `---yaml`  | `---`   | YAML   |
| `---toml`  | `---`   | TOML   |
| `---json`  | `---`   | JSON   |
| `+++`      | `+++`   | TOML   |
| `{`        | `}`     | JSON   |

Trailing whitespace on sentinel lines is ignored. An unrecognised `---<qualifier>` returns an error (e.g. `---yml` is caught as a typo). A missing closing sentinel is also an error — silently treating the remainder as body risks leaking private metadata fields into published content.

If no recognised opening sentinel is found, `meta` is nil and `body` is the full input.

For the full list of front matter formats and their variations, see [front-matter.md](front-matter.md).
