module github.com/client9/tojson/benchmarks

go 1.26.2

require (
	github.com/BurntSushi/toml v1.6.0
	github.com/client9/tojson v0.0.0
	github.com/goccy/go-yaml v1.19.2
	gopkg.in/yaml.v3 v3.0.1
	sigs.k8s.io/yaml v1.6.0
)

require (
	github.com/pelletier/go-toml/v2 v2.3.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
)

replace github.com/client9/tojson => ../
