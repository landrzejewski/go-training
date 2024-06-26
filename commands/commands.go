/*
Zaimplementuj następujące polecenia systemowe w go:

echo - drukuje tekst podany jako argument na standardowym wyjściu
cat  - drukuje zawartość wskazanych plików na standardowym wyjściu,
       zezwala na opcjonalne numerowanie wierszy (przełącznik -n),
	   numerowanie wierszy można wyłączyć dla pustych wierszy (przełącznik -nb)
find - przeszukuje i drukuje ścieżki plików i/lub katalogów, których nazwy pasują do wskazanego wzorca i typu,
       dozwolone typy to plik, katalog lub link symboliczny
grep - wyszukuje i drukuje wiersze zawierające wskazany tekst/wzorzec ze wskazanych plików/ścieżek
*/

package commands

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func Echo() {
	if len(os.Args) > 1 {
		args := os.Args[1:]
		output := strings.Join(args, " ")
		fmt.Println(output)
	}
}

type printer interface {
	print(*int, string)
}

type defaultPrinter struct {
}

func (p *defaultPrinter) print(lineNumber *int, line string)  {
	fmt.Println(line)
}

type numberingPrinter struct {
}

func (p *numberingPrinter) print(lineNumber *int, line string)  {
	fmt.Printf("%6d  %s\n", *lineNumber, line)
	*lineNumber++
}

type numberingWithoutEmptyLinesPrinter struct {
}

func (p *numberingWithoutEmptyLinesPrinter) print(lineNumber *int, line string)  {
	if line != "" {
		fmt.Printf("%6d  %s\n", *lineNumber, line)
		*lineNumber++
	} else {
		fmt.Println(line)
	}
}

func createPrint(numberLines, numberNonEmptyLines bool) printer {
	if numberLines {
		return &numberingPrinter{}
	} else if numberNonEmptyLines {
		return &numberingWithoutEmptyLinesPrinter{}
	} 
	return &defaultPrinter{}
}


// type printer func(*int, string)

// func createPrintFn(numberLines, numberNonEmptyLines bool) printer {
// 	var printFn printer

// 	if numberLines {
// 		printFn = func( lineNumber *int, line string) {
// 			fmt.Printf("%6d  %s\n", *lineNumber, line)
// 			*lineNumber++
// 		}
// 	} else if numberNonEmptyLines {
// 		printFn = func( lineNumber *int, line string) {
// 			if line != "" {
// 				fmt.Printf("%6d  %s\n", *lineNumber, line)
// 				*lineNumber++
// 			} else {
// 				fmt.Println(line)
// 			}
// 		}
// 	} else {
// 		printFn = func(_ *int, line string) {
// 			fmt.Println(line)
// 		}
// 	}
// 	return printFn
// }

func catFile(filename string, numberLines, numberNonEmptyLines bool) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 1

	// printFn := createPrintFn(numberLines, numberNonEmptyLines)
	printer := createPrint(numberLines, numberNonEmptyLines)


	for scanner.Scan() {
		line := scanner.Text()
		printer.print(&lineNumber, line)
		// printFn(&lineNumber, line)
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
		err := catFile(file, *numberLines, *numberNonEmptyLines)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", file, err)
		}
	}
}

func onElement(fileType, pattern *string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.Contains(path, *pattern) {
			return nil
		}

		switch *fileType {
		case "file":
			if info.Mode().IsRegular() {
				fmt.Println(path)
			}
		case "dir":
			if info.IsDir() {
				fmt.Println(path)
			}
		case "symlink":
			if info.Mode()&os.ModeSymlink != 0 {
				fmt.Println(path)
			}
		default:
			fmt.Println("Invalid file type. Allowed types are: file, dir, symlink")
			return fmt.Errorf("invalid file type: %s", *fileType)
		}

		return nil
	}
}

func Find() {
	path := flag.String("p", "", "Start path")
	pattern := flag.String("n", "", "Pattern to match")
	fileType := flag.String("t", "", "Type of file to match (file, dir, symlink)")

	flag.Parse()

	if *pattern == "" || *fileType == "" {
		fmt.Println("Usage: find [-p <start path>] -n <pattern> -t <file|dir|symlink>")
		return
	}

	if *path == "" {
		*path = "."
	}

	err := filepath.Walk(*path, onElement(fileType, pattern))

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func grep(pattern, path string) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Fatalf("Failed to compile regex pattern: %v", err)
	}

	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			log.Printf("Failed to open file %s: %v", path, err)
			return nil
		}
		defer file.Close()

		lineNumber := 0
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			lineNumber++
			if re.MatchString(line) {
				fmt.Printf("%s (line: %d): %s\n", path, lineNumber, line)
			}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("Error reading file %s: %v", path, err)
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Error walking the path %s: %v", path, err)
	}
}

func Grep() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: grep <pattern> <path>")
		os.Exit(1)
	}

	pattern := os.Args[1]
	path := os.Args[2]

	grep(pattern, path)
}