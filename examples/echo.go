package examples

import (
	"fmt"
	"os"
	"strings"
)

// Echo prints the program arguments separated by single spaces, followed by a
// newline - like echo(1), it prints a bare newline when there are no arguments.
func Echo() {
	fmt.Println(strings.Join(os.Args[1:], " "))
}
