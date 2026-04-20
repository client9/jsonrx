package benchmarks

import (
	_ "embed"
	"encoding/json"
	"testing"

	goyaml "github.com/goccy/go-yaml"
	"gopkg.in/yaml.v3"
	sigsyaml "sigs.k8s.io/yaml"

	"github.com/client9/tojson"
)

//go:embed testdata/frontmatter1.yml
var frontmatter1YAML string

func BenchmarkFromYAML(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		raw, err := tojson.FromYAML(frontmatter1YAML)
		if err != nil {
			b.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkYAMLv3ToMap(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var m map[string]any
		if err := yaml.Unmarshal([]byte(frontmatter1YAML), &m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGoccyGoYAMLToMap(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var m map[string]any
		if err := goyaml.Unmarshal([]byte(frontmatter1YAML), &m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSigsK8sYAMLToMap(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var m map[string]any
		if err := sigsyaml.Unmarshal([]byte(frontmatter1YAML), &m); err != nil {
			b.Fatal(err)
		}
	}
}
