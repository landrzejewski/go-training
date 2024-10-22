package concurrency

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func FindFiles() {
	path := "."

	filesChannel := make(chan string)
	filesWithExtensionChannel := make(chan string)
	filesWithContentChannel := make(chan string)

	go findFiles(path, filesChannel)
	go filterByExtension(filesChannel, filesWithExtensionChannel, ".go")
	go filterByContent(filesWithExtensionChannel, filesWithContentChannel, "package concurrency")

	for file := range filesWithContentChannel {
		fmt.Println(file)
	}
}

func findFiles(root string, filesChannel chan<- string) {
	defer close(filesChannel)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			filesChannel <- path
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error walking the path %v: %v\n", root, err)
		return
	}
}

func filterByExtension(filesChannel <-chan string, filesWithExtensionChannel chan<- string, extension string) {
	defer close(filesWithExtensionChannel)
	for file := range filesChannel {
		if strings.HasSuffix(file, extension) {
			filesWithExtensionChannel <- file
		}
	}
}

func filterByContent(filesWithExtensionChannel <-chan string, filesWithContentChannel chan<- string, expectedContent string) {
	defer close(filesWithContentChannel)
	for file := range filesWithExtensionChannel {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Error reading file %v: %v\n", file, err)
			continue
		}
		if strings.Contains(string(content), expectedContent) {
			filesWithContentChannel <- file
		}
	}
}
