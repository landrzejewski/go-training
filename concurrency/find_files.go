package concurrency

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func FindFiles() {
	files := make(chan string, 10)
	filesWithExtension := make(chan string)
	filesWithContent := make(chan string)

	go findFiles(".\\common", files)
	go findFiles(".\\concurrency", files)

	go filterByExtension(files, filesWithExtension, ".go")
	go filterByContent(filesWithExtension, filesWithContent, "package concurrency")

	done := false
	for !done {
		select {
		case file := <-filesWithContent:
			fmt.Println(file)
		case <-time.After(1 * time.Second):
			done = true
		}
	}

	close(files)
	close(filesWithExtension)
	close(filesWithContent)
}

func findFiles(path string, files chan<- string) {
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files <- path
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}

func filterByExtension(files <-chan string, filesWithExtension chan<- string, extension string) {
	for file := range files {
		if strings.HasSuffix(file, extension) {
			filesWithExtension <- file
		}
	}
}

func filterByContent(filesWithExtension <-chan string, filesWithContent chan<- string, text string) {
	for file := range filesWithExtension {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		if strings.Contains(string(content), text) {
			filesWithContent <- file
		}
	}
}
