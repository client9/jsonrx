package tojson_test

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/client9/tojson"
)

func ExampleFromYAML() {
	type Config struct {
		Name   string `json:"name"`
		Port   int    `json:"port"`
		Active bool   `json:"active"`
	}

	src := []byte("name: demo\nport: 8080\nactive: true\n")

	raw, err := tojson.FromYAML(src)
	if err != nil {
		panic(err)
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", cfg)
	// Output:
	// {Name:demo Port:8080 Active:true}
}

func ExampleFromJSONVariant() {
	src := []byte(`
{
  // comments are allowed
  unquoted: 'value',
  hex: 0x2a,
  trailing: [1, 2, 3,],
}
`)

	raw, err := tojson.FromJSONVariant(src)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(raw))
	// Output:
	// {"unquoted":"value","hex":42,"trailing":[1,2,3]}
}

func ExampleParseError() {
	_, err := tojson.FromJSONVariant([]byte("{ unclosed: [1, 2, }"))
	if err != nil {
		var pe *tojson.ParseError
		if errors.As(err, &pe) {
			fmt.Printf("line %d, column %d: %s\n", pe.Line, pe.Column, pe.Message)
		}
	}
	// Output:
	// line 1, column 20: unmatched object end, level=2, stack="{["
}
