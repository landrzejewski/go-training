/*
cat  - drukuje zawartość wskazanych plików na standardowym wyjściu,
zezwala na opcjonalne numerowanie wierszy (przełącznik -n),
numerowanie wierszy można wyłączyć dla pustych wierszy (przełącznik -nb)
*/

package exercises

import (
	"bufio"
	"flag"
	"fmt"
	"os"
)

type printer func(*int, string)

func createPrinter(numberLines, numberNonEmptyLines bool) printer {
	var printFn printer
	switch {
	case numberLines:
		printFn = func(lineNumber *int, line string) {
			fmt.Printf("%6d  %s\n", *lineNumber, line)
			*lineNumber++
		}
	case numberNonEmptyLines:
		printFn = func(lineNumber *int, line string) {
			if line != "" {
				fmt.Printf("%6d  %s\n", *lineNumber, line)
				*lineNumber++
			} else {
				fmt.Println(line)
			}
		}
	default:
		printFn = func(_ *int, line string) {
			fmt.Println(line)
		}
	}
	return printFn
}

func cat(filename string, numberLines, numberNonEmptyLines bool) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	lineNumber := 1
	printFn := createPrinter(numberLines, numberNonEmptyLines)
	for scanner.Scan() {
		line := scanner.Text()
		printFn(&lineNumber, line)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func Cat() {
	numberLines := flag.Bool("n", false, "Number the output lines")
	numberNonEmptyLines := flag.Bool("nb", false, "Number the output lines, but not empty lines")
	flag.Parse()
	files := flag.Args()

	if len(files) == 0 || (*numberLines && *numberNonEmptyLines) {
		fmt.Fprintln(os.Stderr, "Usage: cat [-n|-nb] [file ...]")
		os.Exit(1)
	}

	for _, file := range files {
		fmt.Printf("File: %s\n", file)
		err := cat(file, *numberLines, *numberNonEmptyLines)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", file, err)
		}
		fmt.Println()
	}
}
