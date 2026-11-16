package examples

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
)

// Cat prints the contents of the files named on the command line, optionally
// numbering the lines (-n) or only the non-empty lines (-nb).
//
// An unreadable file is reported and skipped; the remaining files are still
// printed and the process exits with status 1 at the end.
func Cat() {
	// A private FlagSet instead of flag.CommandLine: every demo in this package
	// parses its own flags, and registering "-n" twice on the global set panics
	// with "flag redefined".
	flags := flag.NewFlagSet("cat", flag.ExitOnError)
	numberLines := flags.Bool("n", false, "Number lines")
	numberNonEmptyLines := flags.Bool("nb", false, "Number non-empty lines")
	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	paths := flags.Args()

	if len(paths) == 0 || (*numberLines && *numberNonEmptyLines) {
		// A usage error goes to stderr and exits non-zero - otherwise a script
		// calling cat cannot tell a mistake from a correct, empty result.
		fmt.Fprintln(os.Stderr, "Usage: cat [-n|-nb] path, path ...")
		os.Exit(2)
	}

	// One printer for the whole run: numbering is continuous across all files,
	// just like cat(1).
	printerFn := printerFactory(*numberLines, *numberNonEmptyLines)

	failed := false
	for _, path := range paths {
		fmt.Printf("File: %s\n", path)
		if err := cat(path, printerFn); err != nil {
			log.Printf("error reading %q: %v", path, err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

// printer receives just the line - the counter is internal state of the
// particular implementation. Previously the counter lived in cat and advanced
// for EVERY line, with -nb merely hiding the number on empty lines. For the
// input "a\n\nb\n" that produced 1, (empty), 3 instead of the correct
// 1, (empty), 2.
type printer func(string)

func printerFactory(numberLines, numberNonEmptyLines bool) printer {
	lineNumber := 0
	switch {
	case numberLines:
		return func(line string) {
			lineNumber++
			fmt.Printf("%6d: %s\n", lineNumber, line)
		}
	case numberNonEmptyLines:
		return func(line string) {
			if line == "" {
				fmt.Println(line)
				return
			}
			lineNumber++
			fmt.Printf("%6d: %s\n", lineNumber, line)
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
	// A close error on a file opened for reading is only worth logging.
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("error closing %q: %v", path, err)
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		printerFn(scanner.Text())
	}
	// Scan also returns false on an I/O error and on a line longer than
	// bufio.MaxScanTokenSize (64 KiB) - without this check truncated output would
	// look like a correctly read file.
	return scanner.Err()
}
