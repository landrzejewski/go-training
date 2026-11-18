package solutions

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// ---- Exercise 13.1: strings --------------------------------------------------------------------

// ParseKeyValues parses "key=value" pairs separated by ";" into a map. Spaces around keys and
// values are trimmed. Parts without "=" or with an empty key are skipped.
func ParseKeyValues(s string) map[string]string {
	out := make(map[string]string)
	for _, part := range strings.Split(s, ";") {
		key, value, found := strings.Cut(part, "=") // split once, at the first "="
		key = strings.TrimSpace(key)
		if !found || key == "" {
			continue
		}
		out[key] = strings.TrimSpace(value)
	}
	return out
}

// ---- Exercise 13.2: JSON and files -------------------------------------------------------------

// Book is stored as JSON. The tags choose the JSON names; omitempty leaves out empty Tags.
type Book struct {
	Title  string   `json:"title"`
	Author string   `json:"author"`
	Year   int      `json:"year"`
	Tags   []string `json:"tags,omitempty"`
}

// SaveBooks writes books to the file at path as indented JSON.
func SaveBooks(path string, books []Book) error {
	data, err := json.MarshalIndent(books, "", "  ")
	if err != nil {
		return fmt.Errorf("save books: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("save books: %w", err)
	}
	return nil
}

// LoadBooks reads books saved by SaveBooks. An error from a missing file wraps fs.ErrNotExist.
func LoadBooks(path string) ([]Book, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load books: %w", err)
	}
	var books []Book
	if err := json.Unmarshal(data, &books); err != nil { // Unmarshal needs a pointer
		return nil, fmt.Errorf("load books: %w", err)
	}
	return books, nil
}

// ---- Exercise 13.3: time -----------------------------------------------------------------------

// LogEntry is one parsed log line.
type LogEntry struct {
	Time    time.Time
	Level   string
	Message string
}

// logLayout is Go's reference date (Mon Jan 2 15:04:05 2006) written the way the log writes dates.
const logLayout = "2006-01-02 15:04:05"

// ParseLogLine parses a line such as "2026-03-14 09:30:00 ERROR disk full".
func ParseLogLine(line string) (LogEntry, error) {
	parts := strings.SplitN(line, " ", 4) // date, time, level, and the rest is the message
	if len(parts) < 4 {
		return LogEntry{}, fmt.Errorf("parse log line %q: want date, time, level and message", line)
	}
	t, err := time.Parse(logLayout, parts[0]+" "+parts[1])
	if err != nil {
		return LogEntry{}, fmt.Errorf("parse log line: %w", err)
	}
	return LogEntry{Time: t, Level: parts[2], Message: parts[3]}, nil
}
