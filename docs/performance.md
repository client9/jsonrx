# Performance

- Generated using `make compare`
- Benchmarks were run in April 2026 on an Apple M4 using Go 1.26.2
- Inputs were typical frontmatter-style documents from markdown content.
- All libraries were benchmarked by decoding into `map[string]any`.
- For `tojson`, that means converting the input to JSON and then decoding with `encoding/json`.

## JSON Variants

Summary: Parsing JSON variants adds about 25% overhead over pure `encoding/json`, and uses the same number of allocations.

- There is no standard JSON variant package in Go.
- The small overhead comes from scanning the document to strip/normalize JSON5 features before handing off to `encoding/json`.

| Package | Per Call | Memory | Allocations |
|---------------------------------|------------:|------------:|-------------:|
| **tojson.FromJSONVariants**     |  2393 ns/op |   1992 B/op | 49 allocs/op |
| Go `encoding/json`              |  1908 ns/op |   1608 B/op | 47 allocs/op |


## YAML

Summary: 3x-4x faster, and 4-5x less memory.

Restricting objects to have only string keys, and not storing state with aliases and tags really pays off.

| Package | Per Call | Memory | Allocations |
|-------------------------------------------------------------------------|------------:|------------:|--------------:|
| **tojson.FromYAML**                                                     |  2448 ns/op |   2600 B/op |  51 allocs/op |
| [go-yaml/yaml](https://github.com/go-yaml/yaml) v3.0.1                  |  8213 ns/op |  12752 B/op | 170 allocs/op |
| [goccy/go-yaml](https://github.com/goccy/go-yaml) v1.19.2               | 12710 ns/op |  21456 B/op | 488 allocs/op |
| [kubernetes-sigs/yaml](https://github.com/kubernetes-sigs/yaml) v1.6.0  |  9631 ns/op |  13799 B/op | 238 allocs/op |

## TOML

Summary: About 2x less memory. Comparable speed to `pelletier/go-toml`, 2x faster than `BurntSushi/toml`.

- Compared to `BurntSushi/toml`, it's 2x faster and 2x less memory. 
- Compared to `pelletier/go-toml`, about 2x less memory at similar speed.

| Package | Per Call | Memory | Allocations |
|-------------------------------------------------------------------|------------:|----------:|-------------:|
| **tojson.FromTOML**                                               |  2481 ns/op | 2344 B/op | 55 allocs/op |
| [BurntSushi/toml](https://github.com/BurntSushi/toml) v1.6.0      |  5279 ns/op | 5840 B/op | 99 allocs/op |
| [pelletier/go-toml](https://github.com/pelletier/go-toml) v2.3.0  |  2298 ns/op | 4816 B/op | 68 allocs/op |
