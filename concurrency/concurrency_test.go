package concurrency

import (
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

func Test_WaitGroup(t *testing.T) {
	text := "Test text"

	stdOut := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	wg := sync.WaitGroup{}
	wg.Add(1)

	go print(text, &wg)

	wg.Wait()

	w.Close()

	result, _ := io.ReadAll(r)
	output := string(result)
	os.Stdout = stdOut

	if !strings.Contains(output, text) {
		t.Errorf("Expected to find %v", text)
	}

}
