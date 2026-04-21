# Performance

- Tested April 2026, on Apple M4:
- Input is typical frontmatter as found in markdown documents for blog content.
- All converted the bytes to a `map[string]any`.  This packaged used the standard `encoding/json` for marshaling.


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

```
BenchmarkFromTOML-10                      221962              5317 ns/op            2616 B/op         63 allocs/op

BenchmarkBurntSushiTOMLToMap-10           115405             10347 ns/op            5840 B/op         99 allocs/op
BenchmarkPelletierTOMLToMap-10            261597              4533 ns/op            4816 B/op         68 allocs/op
```
