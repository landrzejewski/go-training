package examples

import (
	"fmt"
	"os"
	"strings"
)

func Echo() {
	args := os.Args[1:]
	if len(args) > 0 {
		echo := strings.Join(args, " ")
		fmt.Println(echo)
	}
}
