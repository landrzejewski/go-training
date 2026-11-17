package solutions

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func ExampleParseKeyValues() {
	fmt.Println(ParseKeyValues(" name = Ann; age=31;; city=Warsaw ; broken; =x"))
	// Output: map[age:31 city:Warsaw name:Ann]
}

func ExampleSaveBooks() {
	dir, err := os.MkdirTemp("", "books")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "books.json")

	books := []Book{
		{Title: "The Go Programming Language", Author: "Donovan, Kernighan", Year: 2015, Tags: []string{"go"}},
		{Title: "Dune", Author: "Herbert", Year: 1965},
	}
	if err := SaveBooks(path, books); err != nil {
		fmt.Println(err)
		return
	}
	data, _ := os.ReadFile(path)
	fmt.Println(string(data))

	loaded, err := LoadBooks(path)
	fmt.Printf("%d books, first: %q, err: %v\n", len(loaded), loaded[0].Title, err)

	_, err = LoadBooks(filepath.Join(dir, "missing.json"))
	fmt.Println(errors.Is(err, fs.ErrNotExist))
	// Output:
	// [
	//   {
	//     "title": "The Go Programming Language",
	//     "author": "Donovan, Kernighan",
	//     "year": 2015,
	//     "tags": [
	//       "go"
	//     ]
	//   },
	//   {
	//     "title": "Dune",
	//     "author": "Herbert",
	//     "year": 1965
	//   }
	// ]
	// 2 books, first: "The Go Programming Language", err: <nil>
	// true
}

func ExampleParseLogLine() {
	entry, err := ParseLogLine("2026-03-14 09:30:00 ERROR disk full")
	fmt.Println(entry.Level, "|", entry.Message, "|", entry.Time.Format("Mon, Jan 2 15:04"), "|", err)

	_, err = ParseLogLine("2026-13-01 10:00:00 INFO hello")
	fmt.Println(err)
	_, err = ParseLogLine("hello")
	fmt.Println(err)
	// Output:
	// ERROR | disk full | Sat, Mar 14 09:30 | <nil>
	// parse log line: parsing time "2026-13-01 10:00:00": month out of range
	// parse log line "hello": want date, time, level and message
}
