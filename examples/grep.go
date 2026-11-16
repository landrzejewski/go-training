package examples

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

// Grep prints every line under the given path that matches the regular
// expression: grep pattern path. Files and directories that cannot be read are
// reported and skipped.
func Grep() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: grep pattern path")
	}

	pattern, err := regexp.Compile(os.Args[1])
	if err != nil {
		log.Fatalf("invalid regular expression %q: %v", os.Args[1], err)
	}

	path := os.Args[2]

	// One *regexp.Regexp is shared by every call of the walk function: a Regexp
	// is safe for concurrent use by multiple goroutines, so there is no reason to
	// copy it (Regexp.Copy was deprecated in Go 1.12 for exactly that reason).
	//
	// WalkDir rather than Walk: recommended since Go 1.16, because the callback
	// receives an fs.DirEntry instead of an fs.FileInfo and so avoids a stat(2)
	// call per entry.
	if err := filepath.WalkDir(path, search(pattern)); err != nil {
		log.Fatalf("Error walking on path %s: %v", path, err)
	}
}

func search(pattern *regexp.Regexp) fs.WalkDirFunc {
	return func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			// Skip what we cannot read rather than aborting the whole walk.
			log.Printf("skipping %q: %v", path, err)
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			// log.Fatalf here used to call os.Exit(1): it skipped every defer
			// (including closing the file), made the return below dead code, and
			// aborted the entire run because of one unreadable file. Return nil
			// to skip the file and keep going.
			log.Printf("skipping %q: %v", path, err)
			return nil
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Printf("error closing %q: %v", path, err)
			}
		}()

		scanner := bufio.NewScanner(file)
		lineNumber := 0
		for scanner.Scan() {
			line := scanner.Text()
			lineNumber++
			if pattern.MatchString(line) {
				fmt.Printf("%s (line: %d): %s\n", path, lineNumber, line)
			}
		}
		// A scanner error (I/O, or a line longer than 64 KiB - easy to hit in a
		// binary file) concerns only this file. Returning it from a WalkDirFunc
		// would abort the whole walk, so - consistently with the other errors
		// above - it is logged and the walk continues.
		if err := scanner.Err(); err != nil {
			log.Printf("error reading %q: %v", path, err)
		}
		return nil
	}
}
