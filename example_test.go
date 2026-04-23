package tojson_test

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/client9/tojson"
)

func ExampleFromYAML() {
	type Article struct {
		Title  string `json:"title"`
		Author string `json:"author"`
		Draft  bool   `json:"draft"`
	}

	src := []byte("title: hello-world\nauthor: alice\ndraft: false\n")

	raw, err := tojson.FromYAML(src)
	if err != nil {
		panic(err)
	}

	var article Article
	if err := json.Unmarshal(raw, &article); err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", article)
	// Output:
	// {Title:hello-world Author:alice Draft:false}
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
