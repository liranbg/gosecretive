# gosecretive

`gosecretive` is a library for scrubbing sensitive _string_ data from `go` objects,
by placing a reference value to the original value in the scrubbed object.

`gosecretive` do not use struct field tags but list of field paths, delimited by `/`, to allow
field scrubbing in nested structs.

## Get

```bash
go get github.com/nuclio/gosecretive@latest
```

## Usage

A complete roundtrip for scrubbing and restoring sensitive data:

```go
package main

import (
	"fmt"

	"github.com/nuclio/gosecretive"
)

type SomeStruct struct {
	NotASecret string                 `json:"not_a_secret,omitempty"`
	Secret     string                 `json:"secret,omitempty"`
	Map        map[string]interface{} `json:"map,omitempty"`
	List       []interface{}          `json:"list,omitempty"`
}

func main() {
	s := SomeStruct{
		NotASecret: "not a secret",
		Secret:     "secret",
		Map: map[string]interface{}{
			"secret": "secret",
		},
		List: []interface{}{
			"not-a-secret",
			"secret",
		},
	}

	// {"secret":"$ref:/Secret","map":{"secret":"$ref:/Map/secret"},"list":["not-a-secret","secret"]}
	scrubbed, secrets := gosecretive.Scrub(&s, func(fieldPath string, valueToScrub interface{}) *string {
		for _, fieldPathToScrub := range []string{
			"/Secret",
			"/Map/secret",
			"/List[1]",
		} {
			if fieldPath == fieldPathToScrub {

				// scrub the value, leave a placeholder to allow restoring later on
				secretKey := fmt.Sprintf("$ref-%s", fieldPath)
				return &secretKey
			}
		}

		// do not scrub
		return nil
	})

	// {NotASecret:not a secret Secret:$ref-/Secret Map:map[secret:$ref-/Map/secret] List:[not-a-secret $ref-/List[1]]}
	fmt.Printf("Scrubbed %+v\n", *scrubbed.(*SomeStruct))

	// map[$ref-/List[1]:secret $ref-/Map/secret:secret $ref-/Secret:secret]
	fmt.Printf("Secrets %s\n", secrets)

	// now restore object using secrets
	restored := gosecretive.Restore(scrubbed, secrets)

	// {NotASecret:not a secret Secret:secret Map:map[secret:secret] List:[not-a-secret secret]}
	fmt.Printf("Restored %+v\n", *restored.(*SomeStruct))
}
```

## License

[Apache 2.0](LICENSE)

## Acknowledgements

This project was inspired by [go-scrub](https://github.com/ssrathi/go-scrub). The main difference made on this
project is the ability to scrub fields that are part of nested structs, by using a field path.
