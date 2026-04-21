# Performance

- Generated using `make compare`
- Benchmarks were run in April 2026 on an Apple M4 using Go 1.26.2
- Inputs were typical frontmatter-style documents from markdown content.
- All libraries were benchmarked by decoding into `map[string]any`.
- For `tojson`, that means converting the input to JSON and then decoding with `encoding/json`.

## YAML

Summary: 3x-5x faster, and used 3-5x less memory.

```
BenchmarkFromYAML-10                      210936              5454 ns/op            3656 B/op         65 allocs/op

BenchmarkYAMLv3ToMap-10                    74374             16080 ns/op           12752 B/op        170 allocs/op
BenchmarkGoccyGoYAMLToMap-10               47630             25073 ns/op           21456 B/op        488 allocs/op
BenchmarkSigsK8sYAMLToMap-10               62505             19078 ns/op           13799 B/op        238 allocs/op
```

## TOML

Summary: Used about 2x less memory.  Performance ranged from 0.8x to 2x.

| Package | Per Call | Memory | Allocations |
|-------------------------------------------------------------------|------------:|----------:|-------------:|
| **tojson.FromTOML**                                               |  5317 ns/op | 2616 B/op | 63 allocs/op |
| [BurntSushi/toml](https://github.com/BurntSushi/toml) v1.6.0      | 10347 ns/op | 5840 B/op | 99 allocs/op |
| [pelletier/go-toml](https://github.com/pelletier/go-toml) v2.3.0  |  4533 ns/op | 4816 B/op | 68 allocs/op |
```
