# Performance

- Generated using `make compare`
- Benchmarks were run in April 2026 on an Apple M4 using Go 1.26.2
- Inputs were typical frontmatter-style documents from markdown content.
- All libraries were benchmarked by decoding into `map[string]any`.
- For `tojson`, that means converting the input to JSON and then decoding with `encoding/json`.

## JSON Variants

Summary: Parsing JSON variants is around 2x slower, and 3x more memory.

It's slower is every number and string needs to be checked and normalized. The whole document is effectively parsed twice.

| Package | Per Call | Memory | Allocations |
|---------------------------------|------------:|------------:|-------------:|
| **tojson.FromJSONVariants**     |  6723 ns/op |   5720 B/op | 70 allocs/op |
| Go `encoding/json`              |  3554 ns/op |   1608 B/op | 47 allocs/op |


## YAML

Summary: 3x-5x faster, and used 3-5x less memory.

Restricting objects to have only string keys, and not storing state with aliases and tags really pays off.

| Package | Per Call | Memory | Allocations |
|-------------------------------------------------------------------------|------------:|------------:|--------------:|
| **tojson.FromYAML**                                                     |  5454 ns/op |   3656 B/op |  65 allocs/op |
| [go-yaml/yaml](https://github.com/go-yaml/yaml) v3.0.1                  | 16080 ns/op |  12752 B/op | 170 allocs/op |
| [goccy/go-yaml](https://github.com/goccy/go-yaml) v1.19.2               | 25073 ns/op |  21456 B/op | 488 allocs/op |
| [kubernetes-sigs/yaml](https://github.com/kubernetes-sigs/yaml) v1.6.0  | 19078 ns/op |  13799 B/op | 238 allocs/op |

## TOML

Summary: Used about 2x less memory.  Performance ranged from 0.8x to 2x.

Compared to `BurntSushi/toml`, it's 2x faster and 2x less memory.  Compared to `pelletier/go-toml` is more mixed.  About 2x less memory, but ran 20% slower.  If you are only parsing TOML, `pelletier/go-toml` is worth evaluating.

| Package | Per Call | Memory | Allocations |
|-------------------------------------------------------------------|------------:|----------:|-------------:|
| **tojson.FromTOML**                                               |  5317 ns/op | 2616 B/op | 63 allocs/op |
| [BurntSushi/toml](https://github.com/BurntSushi/toml) v1.6.0      | 10347 ns/op | 5840 B/op | 99 allocs/op |
| [pelletier/go-toml](https://github.com/pelletier/go-toml) v2.3.0  |  4533 ns/op | 4816 B/op | 68 allocs/op |

