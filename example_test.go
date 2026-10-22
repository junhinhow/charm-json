package charmjson_test

import (
	"fmt"
	"strings"

	charmjson "github.com/junhinhow/charm-json"
)

func ExampleParseJSON() {
	root, err := charmjson.ParseJSON([]byte(`{"name":"charm-json","tags":["json","yaml"]}`))
	if err != nil {
		panic(err)
	}

	for _, row := range root.Flatten() {
		fmt.Printf("%s%s\n", strings.Repeat("  ", row.Depth), row.Display)
	}

	// Output:
	// {2}
	//   name: "charm-json"
	//   tags: [2]
	//     0: "json"
	//     1: "yaml"
}
