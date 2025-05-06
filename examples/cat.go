package examples

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
)

func Cat() {
	numberLines := flag.Bool("n", false, "Number lines")
	numberNonEmptyLines := flag.Bool("nb", false, "Number non empty lines")
	flag.Parse()
	paths := flag.Args()

	if len(paths) == 0 || (*numberLines && *numberNonEmptyLines) {
		fmt.Println("Usege: cat [-n|-nb] path, path ...")
		os.Exit(0)
	}

	printerFn := printerFactory(*numberLines, *numberNonEmptyLines)

	for _, path := range paths {
		fmt.Printf("File: %s\n", path)
		if err := cat(path, printerFn); err != nil {
			log.Fatal("Error reading path: ", path)
		}
	}
}

type printer = func(int, string)

func printerFactory(numberLines, numberNonEmptyLines bool) (printerFn printer) {
	switch {
	case numberLines:
		printerFn = func(lineNumber int, line string) {
			fmt.Printf("%6d: %s\n", lineNumber, line)
		}
	case numberNonEmptyLines:
		printerFn = func(lineNumber int, line string) {
			if line != "" {
				fmt.Printf("%6d: %s\n", lineNumber, line)
			} else {
				fmt.Println(line)
			}
		}
	default:
		printerFn = func(lineNumber int, line string) {
			fmt.Println(line)
		}
	}
	return printerFn
}

func cat(path string, printerFn printer) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func (file *os.File)  {
		if file.Close() != nil {
			log.Fatal("Error closing file: ", file)
		}
	}(file)
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++
		printerFn(lineNumber, line)
	}	
	return nil
}
