// echo - drukuje tekst podany jako argumenty programu na standardowym wyjściu

package exercises

import (
	"fmt"
	"os"
	"strings"
)

func Echo() {
	if len(os.Args) > 1 {
		args := os.Args[1:]
		output := strings.Join(args, " ")
		fmt.Println(output)
	}
}
