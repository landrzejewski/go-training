package exercises

import (
	"flag"
	"fmt"
	"path/filepath"
)

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
