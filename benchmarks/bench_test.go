package benchmarks

import (
	"encoding/json"
	"testing"

	burntsushitoml "github.com/BurntSushi/toml"
	pelletiertoml "github.com/pelletier/go-toml/v2"
	goyaml "github.com/goccy/go-yaml"
	"gopkg.in/yaml.v3"
	sigsyaml "sigs.k8s.io/yaml"

	"github.com/client9/tojson"
)

const frontmatter1YAML = `date: 2024-02-02T04:14:54-08:00
draft: false
genres:
- mystery
- romance
tags:
- red
- blue
title: Example
weight: 10
params:
  author: John Smith
`

const frontmatter1TOML = `date = 2024-02-02T04:14:54-08:00
draft = false
genres = ['mystery', 'romance']
tags = ['red', 'blue']
title = 'Example'
weight = 10
[params]
  author = 'John Smith'
`

func BenchmarkFromYAML(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		raw, err := tojson.FromYAML([]byte(frontmatter1YAML))
		if err != nil {
			b.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFromTOML(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		raw, err := tojson.FromTOML([]byte(frontmatter1TOML))
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

func BenchmarkBurntSushiTOMLToMap(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var m map[string]any
		if _, err := burntsushitoml.Decode(frontmatter1TOML, &m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPelletierTOMLToMap(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var m map[string]any
		if err := pelletiertoml.Unmarshal([]byte(frontmatter1TOML), &m); err != nil {
			b.Fatal(err)
		}
	}
}
