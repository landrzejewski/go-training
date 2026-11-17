// Package search is the reference solution for Project 4 in basic_exercises.md: a small
// full-text search engine.
//
// It splits documents into words (tokens), runs them through a chain of filters, builds an
// inverted index (word -> the documents that contain it), answers queries such as
// "go concurrency -java" ranked by relevance, and saves and reloads its documents as JSON.
//
// An Index is safe for concurrent use: searches share a read lock, adding takes the write lock.
package search

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"maps"
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// ---- Errors ------------------------------------------------------------------------------------

// Sentinel errors. Add wraps the document errors in a *DocError; check them with errors.Is.
var (
	ErrEmptyID     = errors.New("document id is empty")
	ErrDuplicateID = errors.New("duplicate document id")
	ErrEmptyQuery  = errors.New("query has no search terms")
)

// DocError says which document could not be added, and why.
type DocError struct {
	ID  string
	Err error
}

func (e *DocError) Error() string { return fmt.Sprintf("document %q: %v", e.ID, e.Err) }

// Unwrap lets errors.Is(err, ErrDuplicateID) see through the DocError.
func (e *DocError) Unwrap() error { return e.Err }

// ---- Analysis: from text to tokens -------------------------------------------------------------

// TokenFilter changes a token or drops it (keep == false). An Analyzer runs its filters in
// order, and the first filter that drops a token ends the chain for that token.
type TokenFilter interface {
	Filter(token string) (result string, keep bool)
}

// FilterFunc turns a plain function into a TokenFilter. A method on a FUNCTION type: the same
// trick as http.HandlerFunc in the standard library.
type FilterFunc func(token string) (string, bool)

func (f FilterFunc) Filter(token string) (string, bool) { return f(token) }

// Lowercase makes matching case-insensitive. strings.ToLower knows Unicode: "ŻÓŁW" -> "żółw".
var Lowercase TokenFilter = FilterFunc(func(t string) (string, bool) {
	return strings.ToLower(t), true
})

// MinLength drops tokens with fewer than n characters - runes, not bytes, so "ął" has 2.
func MinLength(n int) TokenFilter {
	return FilterFunc(func(t string) (string, bool) {
		return t, utf8.RuneCountInString(t) >= n
	})
}

// StopWords drops the given words. The set is built once, here; the returned filter is a
// closure that keeps using it.
func StopWords(words ...string) TokenFilter {
	stop := NewSet[string]()
	for _, w := range words {
		stop.Add(strings.ToLower(w))
	}
	return FilterFunc(func(t string) (string, bool) {
		return t, !stop.Has(t)
	})
}

// Analyzer splits text into words and runs every word through its filters. Documents and
// queries go through the same Analyzer, so "Go" in a query finds "go" in a document.
type Analyzer struct {
	filters []TokenFilter
}

// NewAnalyzer returns an Analyzer with the given filters, applied in order.
func NewAnalyzer(filters ...TokenFilter) *Analyzer {
	return &Analyzer{filters: filters}
}

// DefaultAnalyzer lowercases, drops one-letter words and some common English words.
func DefaultAnalyzer() *Analyzer {
	return NewAnalyzer(
		Lowercase, // first, so that the stop words below match "The" as well as "the"
		MinLength(2),
		StopWords("an", "and", "are", "for", "in", "is", "of", "on", "or", "the", "to", "with"),
	)
}

// Tokens yields the words of text after filtering. A word is a run of letters and digits, so
// "Go's" splits into "Go" and "s". The iterator is lazy: strings.FieldsFuncSeq (Go 1.24) finds
// one word at a time, and a break in the caller's loop stops the work.
func (a *Analyzer) Tokens(text string) iter.Seq[string] {
	isSeparator := func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }
	return func(yield func(string) bool) {
		for word := range strings.FieldsFuncSeq(text, isSeparator) {
			if tok, keep := a.apply(word); keep && !yield(tok) {
				return
			}
		}
	}
}

func (a *Analyzer) apply(tok string) (string, bool) {
	for _, f := range a.filters {
		var keep bool
		if tok, keep = f.Filter(tok); !keep {
			return "", false
		}
	}
	return tok, true
}

// ---- A generic set -----------------------------------------------------------------------------

// Set is a collection of unique values (as in exercise 10.3), used here to combine the
// results of several search terms.
type Set[T comparable] struct {
	items map[T]struct{}
}

// NewSet returns a set holding the given items.
func NewSet[T comparable](items ...T) *Set[T] {
	s := &Set[T]{items: make(map[T]struct{}, len(items))}
	for _, v := range items {
		s.Add(v)
	}
	return s
}

func (s *Set[T]) Add(v T) { s.items[v] = struct{}{} }

func (s *Set[T]) Has(v T) bool {
	_, ok := s.items[v]
	return ok
}

func (s *Set[T]) Len() int { return len(s.items) }

// Intersect returns a NEW set with the values that are in both s and other.
func (s *Set[T]) Intersect(other *Set[T]) *Set[T] {
	out := NewSet[T]()
	for v := range s.items {
		if other.Has(v) {
			out.Add(v)
		}
	}
	return out
}

// Difference returns a NEW set with the values of s that are not in other.
func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	out := NewSet[T]()
	for v := range s.items {
		if !other.Has(v) {
			out.Add(v)
		}
	}
	return out
}

// All yields the values in no particular order.
func (s *Set[T]) All() iter.Seq[T] { return maps.Keys(s.items) }

// ---- The index ---------------------------------------------------------------------------------

// Document is what gets indexed.
type Document struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// Result is one search hit.
type Result struct {
	ID    string
	Title string
	Score int
}

// Index is an inverted index: for every term, the documents that contain it and how often.
type Index struct {
	mu       sync.RWMutex
	analyzer *Analyzer
	docs     map[string]Document
	postings map[string]map[string]int // term -> document ID -> weighted count
}

// NewIndex returns an empty index. A nil analyzer means DefaultAnalyzer().
func NewIndex(a *Analyzer) *Index {
	if a == nil {
		a = DefaultAnalyzer()
	}
	return &Index{
		analyzer: a,
		docs:     make(map[string]Document),
		postings: make(map[string]map[string]int),
	}
}

// termCounts analyses one document: a word in the title counts 2, a word in the body 1. It
// reads no shared state (the analyzer never changes), so it may run in many goroutines.
func (ix *Index) termCounts(doc Document) map[string]int {
	counts := make(map[string]int)
	for tok := range ix.analyzer.Tokens(doc.Title) {
		counts[tok] += 2
	}
	for tok := range ix.analyzer.Tokens(doc.Body) {
		counts[tok]++
	}
	return counts
}

// Add indexes one document. Its ID must not be blank or already in use.
func (ix *Index) Add(doc Document) error {
	if strings.TrimSpace(doc.ID) == "" {
		return &DocError{ID: doc.ID, Err: ErrEmptyID}
	}
	counts := ix.termCounts(doc) // the slow part runs BEFORE taking the lock

	ix.mu.Lock()
	defer ix.mu.Unlock()
	return ix.insert(doc, counts)
}

// AddAll indexes many documents. The analysis runs on a pool of workers; the documents are
// then inserted in input order. A bad document does not stop the others: every failure is
// returned, joined into one error.
func (ix *Index) AddAll(docs []Document, workers int) error {
	counts := make([]map[string]int, len(docs)) // one slot per document: no mutex needed
	jobs := make(chan int)

	var wg sync.WaitGroup
	for range max(workers, 1) {
		wg.Go(func() {
			for i := range jobs {
				counts[i] = ix.termCounts(docs[i])
			}
		})
	}
	for i := range docs {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	ix.mu.Lock()
	defer ix.mu.Unlock()
	var errs []error
	for i, doc := range docs {
		if strings.TrimSpace(doc.ID) == "" {
			errs = append(errs, &DocError{ID: doc.ID, Err: ErrEmptyID})
			continue
		}
		if err := ix.insert(doc, counts[i]); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...) // nil when every document was added
}

// insert stores an analysed document. The caller must hold the write lock.
func (ix *Index) insert(doc Document, counts map[string]int) error {
	if _, exists := ix.docs[doc.ID]; exists {
		return &DocError{ID: doc.ID, Err: ErrDuplicateID}
	}
	ix.docs[doc.ID] = doc
	for term, n := range counts {
		if ix.postings[term] == nil {
			ix.postings[term] = make(map[string]int) // the inner map is created on first use
		}
		ix.postings[term][doc.ID] = n
	}
	return nil
}

// Search returns the documents that contain EVERY term of the query and none of the terms
// written with a leading "-", best first. The score of a document is the sum of the weighted
// counts of the query terms in it; equal scores are ordered by ID.
func (ix *Index) Search(query string) ([]Result, error) {
	include, exclude := ix.parseQuery(query)
	if len(include) == 0 {
		return nil, fmt.Errorf("search %q: %w", query, ErrEmptyQuery)
	}

	ix.mu.RLock() // many searches may run at once
	defer ix.mu.RUnlock()

	matches := ix.docsWith(include[0])
	for _, term := range include[1:] {
		matches = matches.Intersect(ix.docsWith(term)) // AND
	}
	for _, term := range exclude {
		matches = matches.Difference(ix.docsWith(term)) // AND NOT
	}

	results := make([]Result, 0, matches.Len()) // not nil: "no hits" is an empty list, not an error
	for id := range matches.All() {
		score := 0
		for _, term := range include {
			score += ix.postings[term][id]
		}
		results = append(results, Result{ID: id, Title: ix.docs[id].Title, Score: score})
	}
	slices.SortFunc(results, func(a, b Result) int {
		return cmp.Or(cmp.Compare(b.Score, a.Score), cmp.Compare(a.ID, b.ID)) // score desc, then ID
	})
	return results, nil
}

// parseQuery splits a query into the terms to include and the terms to exclude ("-java").
// Query words go through the analyzer too, so stop words disappear and case does not matter.
func (ix *Index) parseQuery(query string) (include, exclude []string) {
	for _, word := range strings.Fields(query) {
		target := &include // a POINTER to the slice we append to
		if rest, isExcluded := strings.CutPrefix(word, "-"); isExcluded {
			word, target = rest, &exclude
		}
		for tok := range ix.analyzer.Tokens(word) {
			*target = append(*target, tok)
		}
	}
	return include, exclude
}

// docsWith returns the IDs of the documents containing term. The caller must hold a lock.
func (ix *Index) docsWith(term string) *Set[string] {
	s := NewSet[string]()
	for id := range ix.postings[term] { // ranging over a missing (nil) inner map: zero iterations
		s.Add(id)
	}
	return s
}

// Len returns the number of documents.
func (ix *Index) Len() int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return len(ix.docs)
}

// Terms yields every indexed term with the number of documents containing it, in
// alphabetical order. It works on a snapshot, so the loop body may use the index.
func (ix *Index) Terms() iter.Seq2[string, int] {
	return func(yield func(string, int) bool) {
		ix.mu.RLock()
		docCounts := make(map[string]int, len(ix.postings))
		for term, docs := range ix.postings {
			docCounts[term] = len(docs)
		}
		ix.mu.RUnlock()

		for _, term := range slices.Sorted(maps.Keys(docCounts)) {
			if !yield(term, docCounts[term]) {
				return
			}
		}
	}
}

// Save writes all documents as indented JSON, sorted by ID. The postings are NOT saved: Load
// rebuilds them, which keeps the file small and its format simple.
func (ix *Index) Save(w io.Writer) error {
	ix.mu.RLock()
	docs := slices.Collect(maps.Values(ix.docs))
	ix.mu.RUnlock()
	slices.SortFunc(docs, func(a, b Document) int { return cmp.Compare(a.ID, b.ID) })

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(docs); err != nil {
		return fmt.Errorf("save index: %w", err)
	}
	return nil
}

// Load reads documents written by Save and indexes them with a (nil means DefaultAnalyzer).
func Load(r io.Reader, a *Analyzer, workers int) (*Index, error) {
	var docs []Document
	if err := json.NewDecoder(r).Decode(&docs); err != nil {
		return nil, fmt.Errorf("load index: %w", err)
	}
	ix := NewIndex(a)
	if err := ix.AddAll(docs, workers); err != nil {
		return nil, fmt.Errorf("load index: %w", err)
	}
	return ix, nil
}
