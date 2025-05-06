package examples

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

func Grep() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: grep pattern path")
	}

	pattern, err := regexp.Compile(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid regexp")
	}

	path := os.Args[2]

	err = filepath.Walk(path, search(*pattern)) 

	if err != nil {
		log.Fatalf("Error walking on path %s: %v", path, err)
	}
}

func search(pattern regexp.Regexp) filepath.WalkFunc {
	return  func(path string, info os.FileInfo, err error) error { 
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			log.Fatalf("Failed to open file: %s", path)
			return err
		}
		defer func (file *os.File)  {
			if file.Close() != nil {
				log.Fatalf("Failed to close file: %s", path)
			}
		}(file)
		scanner := bufio.NewScanner(file)
		lineNumber := 0
		for scanner.Scan() {
			line := scanner.Text()
			lineNumber++
			if pattern.MatchString(line) {
				fmt.Printf("%s (line: %d): %s\n", path, lineNumber, line)
			}
		}	
		return err
	}
}
