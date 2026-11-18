package tools

import (
	"fmt"
	"os"
	"strings"
)

func Echo() {
	args := os.Args[1:]
	fmt.Println(strings.Join(args, " "))
}
