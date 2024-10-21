/*
cat  - drukuje zawartość wskazanych plików na standardowym wyjściu,
zezwala na opcjonalne numerowanie wierszy (przełącznik -n),
numerowanie wierszy można wyłączyć dla pustych wierszy (przełącznik -nb)
*/

package exercises

import "flag"

func Cat() {
	numberLines := flag.Bool("n", false, "Number the output lines")
	numberNonEmptyLines := flag.Bool("nb", false, "Number the output lines, but not empty lines")
	flag.Parse()

	files := flag.Args()
}
