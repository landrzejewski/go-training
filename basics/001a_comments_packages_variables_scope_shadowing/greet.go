package main

// This second file exists to show that a package is one namespace shared by all its files:
// main.go calls greet and reads Version without importing anything.
// Only one file of a package should carry the package doc comment (here: main.go).

// Version is exported (upper case), but in package main nobody can import it anyway.
const Version = "1.0"

// greet returns a greeting for name.
//
// This is a doc comment: it sits directly above the declaration, starts with the declared name,
// and may contain paragraphs, [Version]-style links and indented code:
//
//	fmt.Println(greet("Go"))
func greet(name string) string {
	return "Hello, " + name + "!"
}
