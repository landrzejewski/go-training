package tools

import (
	"bufio"
	"flag"
	"fmt"
	"os"
)

func Cat() {
	flags := flag.NewFlagSet("cat", flag.ExitOnError)
	numberLines := flags.Bool("n", false, "Number lines")
	numberNonEmptyLines := flags.Bool("nb", false, "Number non-empty lines")
	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	paths := flags.Args()
	if len(paths) == 0 || *numberLines && *numberNonEmptyLines {
		fmt.Println("Usage: cat [-n|-nb] path path...")
		os.Exit(2)
	}
	printerFn := createPrinter(*numberLines, *numberNonEmptyLines)
	for _, path := range paths {
		fmt.Printf("%s\n", path)
		if err := cat(path, printerFn); err != nil {
			fmt.Println(err)
		}
	}
}

type printer func(string)

func createPrinter(numberLines, numberNonEmptyLines bool) printer {
	lineNumber := 0
	p1 := func(line string) {
		lineNumber++
		fmt.Printf("%6d: %s\n", lineNumber, line)
	}
	switch {
	case numberLines:
		return p1
	case numberNonEmptyLines:
		return func(line string) {
			if line == "" {
				fmt.Println()
				return
			}
			p1(line)
		}
	default:
		return func(line string) {
			fmt.Println(line)
		}
	}
}

func cat(path string, printerFn printer) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		printerFn(scanner.Text())
	}
	return scanner.Err()
}
