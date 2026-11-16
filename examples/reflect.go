package examples

import (
	"fmt"
	"reflect"
)

// Person is a sample struct whose fields carry custom "mymeta" tags; Reflect
// reads them back at run time.
type Person struct {
	Name string `mymeta:"required" training:"required"`
	Age  int    `mymeta:"range=0..150"`
}

// Reflect walks the fields of Person and prints the value of every "mymeta"
// struct tag.
func Reflect() {
	// TypeFor (Go 1.22+) avoids constructing a throwaway Person{} value.
	t := reflect.TypeFor[Person]()

	for i := range t.NumField() {
		f := t.Field(i)
		if tagVal := f.Tag.Get("mymeta"); tagVal != "" {
			fmt.Printf("%s -> %q\n", f.Name, tagVal)
		}
	}
}
