package exercises

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func onElement(fileType, pattern *string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.Contains(info.Name(), *pattern) {
			return nil
		}
		switch *fileType {
		case "file":
			if info.Mode().IsRegular() {
				fmt.Println(path)
			}
		case "dir":
			if info.Mode().IsDir() {
				fmt.Println(path)
			}
		case "symlink":
			if info.Mode()&os.ModeSymlink == os.ModeSymlink {
				fmt.Println(path)
			}
		default:
			fmt.Println("File type not supported")
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
