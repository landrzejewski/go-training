package examples

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func Find() {
	path := flag.String("p", "", "Start path")
	name := flag.String("n", "", "Name to match")
	fileType := flag.String("t", "", "Type to match (file, dir, symlink)")

	flag.Parse()

	if *name == "" || *fileType == "" {
		flag.Usage()
		return
	}

	if *path == "" {
		*path = "."
	}

	if filepath.Walk(*path, onElement(*fileType, *name)) != nil {
		log.Fatalf("Error reading file")
		return
	}
}

func onElement(fileType, name string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.Contains(info.Name(), name) {
			return nil
		}
		switch fileType {
		case fileType:
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
		}
		return nil
	}
}
