package exercises

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

func grep(pattern, path string) {
	regularExpression, err := regexp.Compile(pattern)
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
			if regularExpression.MatchString(line) {
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
		log.Fatalf("Usage: grep <pattern> <path>")
	}
	grep(os.Args[1], os.Args[2])
}
