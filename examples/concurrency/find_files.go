package concurrency

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FindFiles prints every .go file under roots whose content contains pattern.
// With no roots it searches the current directory.
//
// It builds a three-stage pipeline:
//
//	findFiles -> filterByExtension -> filterByContent
//
// The rule this example demonstrates: ONLY the sender closes a channel, and only
// when it is the sole sender. When there are several senders (here: one
// findFiles goroutine per root) we wait for them with a sync.WaitGroup and close
// the channel only once they have all finished.
func FindFiles(pattern string, roots ...string) {
	if len(roots) == 0 {
		roots = []string{"."}
	}

	files := make(chan string, 10)
	filesWithExtension := make(chan string)
	filesWithContent := make(chan string)

	// Stage 1: several senders write to the same channel, so none may close it
	// on its own - a separate goroutine does that after wg.Wait().
	var wg sync.WaitGroup
	for _, root := range roots {
		wg.Add(1)
		go func() {
			defer wg.Done()
			findFiles(root, files)
		}()
	}
	go func() {
		wg.Wait()
		close(files)
	}()

	// Stages 2 and 3: each has exactly one sender, so each closes its own channel.
	go func() {
		defer close(filesWithExtension)
		filterByExtension(files, filesWithExtension, ".go")
	}()
	go func() {
		defer close(filesWithContent)
		filterByContent(filesWithExtension, filesWithContent, pattern)
	}()

	// The loop ends when the last stage closes its channel - no artificial timeout.
	for file := range filesWithContent {
		fmt.Println(file)
	}
}

// findFiles sends the path of every regular file under root. Unreadable
// entries are logged and skipped so that one bad directory does not cut the
// whole search short.
func findFiles(root string, files chan<- string) {
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("skipping %q: %v", path, err)
			return nil
		}
		if entry.Type().IsRegular() {
			files <- path
		}
		return nil
	})
	// Only an error about root itself can get here (every other error is
	// swallowed by the callback above). It ends just this stage; the other roots
	// and the rest of the pipeline are unaffected.
	if err != nil {
		log.Printf("error walking %q: %v", root, err)
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
			log.Printf("skipping %q: %v", file, err)
			continue
		}
		if strings.Contains(string(content), text) {
			filesWithContent <- file
		}
	}
}
