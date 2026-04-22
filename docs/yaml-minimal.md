# Minimal YAML

`tojson.FromYAML` converts a subset of YAML directly to JSON bytes, then hands off to Go's standard `encoding/json` for unmarshalling. No reflection, no intermediate tree — just a single pass from YAML to JSON.

Most real-world YAML is simple: string keys, scalar values, nested maps and lists. `FromYAML` handles all of that. What it skips — anchors, tags, complex keys — most applications never use. 

```go
raw, err := tojson.FromYAML(yamlInput)  // → compact JSON bytes
if err != nil { ... }

var cfg MyConfig
json.Unmarshal(raw, &cfg)               // standard JSON unmarshalling
```

Because the output is plain JSON, you only need `json` struct tags — no separate `yaml` tags, and the full JSON ecosystem (validation, pretty-printing, streaming) works as-is.

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

The benefit being only json struct tags are needed, and can leveage all the JSON unmarhsalling infrastructure.  That makes sense, but this implementation is 7x slower than `tojson`/
.

The concept is described [here](https://web.archive.org/web/20190603050330/http://ghodss.com/2014/the-right-way-to-handle-yaml-in-golang/).
