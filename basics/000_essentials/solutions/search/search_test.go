package search

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
)

// corpus is a small set of documents shared by the tests.
var corpus = []Document{
	{ID: "d1", Title: "Concurrency in Go", Body: "Goroutines and channels make concurrency simple. Go is fun."},
	{ID: "d2", Title: "Generics", Body: "Go generics use type parameters and constraints."},
	{ID: "d3", Title: "Java threads", Body: "Java uses threads for concurrency, Go uses goroutines."},
	{ID: "d4", Title: "Cooking pasta", Body: "Boil water, add salt and pasta."},
	{ID: "d5", Title: "Go channels", Body: "A channel connects goroutines."},
}

// newTestIndex builds an index of the corpus or stops the test.
func newTestIndex(t *testing.T) *Index {
	t.Helper()
	ix := NewIndex(nil)
	if err := ix.AddAll(corpus, 3); err != nil {
		t.Fatalf("AddAll: %v", err)
	}
	return ix
}

func ids(results []Result) []string {
	out := make([]string, 0, len(results))
	for _, r := range results {
		out = append(out, r.ID)
	}
	return out
}

func TestTokens(t *testing.T) {
	a := DefaultAnalyzer()
	tests := []struct {
		text string
		want []string
	}{
		{"Hello, World!", []string{"hello", "world"}},
		{"Go's goroutines & channels", []string{"go", "goroutines", "channels"}}, // "s" is too short
		{"The art of the deal", []string{"art", "deal"}},                         // stop words
		{"ZAŻÓŁĆ gęślą jaźń", []string{"zażółć", "gęślą", "jaźń"}},               // Unicode letters and case
		{"Go 1.27 released", []string{"go", "27", "released"}},                   // "1" is too short
		{"  ...  ", nil},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := slices.Collect(a.Tokens(tt.text))
			if !slices.Equal(got, tt.want) {
				t.Errorf("Tokens(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestTokensStopsOnBreak(t *testing.T) {
	n := 0
	for range DefaultAnalyzer().Tokens(strings.Repeat("word ", 1000)) {
		n++
		if n == 3 {
			break
		}
	}
	if n != 3 {
		t.Errorf("got %d tokens after break, want 3", n)
	}
}

// pluralStripper is a filter of our OWN type: anything with a Filter method is a TokenFilter.
type pluralStripper struct{}

func (pluralStripper) Filter(t string) (string, bool) {
	if len(t) > 3 {
		t = strings.TrimSuffix(t, "s")
	}
	return t, true
}

func TestCustomFilter(t *testing.T) {
	ix := NewIndex(NewAnalyzer(Lowercase, pluralStripper{}))
	if err := ix.AddAll(corpus, 2); err != nil {
		t.Fatal(err)
	}
	results, err := ix.Search("goroutine channel") // singular now matches "goroutines", "channels"
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(results); !slices.Equal(got, []string{"d5", "d1"}) {
		t.Errorf("results = %v, want [d5 d1]", got)
	}
}

func TestSearch(t *testing.T) {
	ix := newTestIndex(t)
	tests := []struct {
		query string
		want  []string
	}{
		{"go", []string{"d1", "d5", "d2", "d3"}},       // d1 has go in title AND body twice
		{"GO Concurrency", []string{"d1", "d3"}},       // AND, and case does not matter
		{"go concurrency -java", []string{"d1"}},       // exclusion
		{"the goroutines", []string{"d1", "d3", "d5"}}, // the stop word is ignored; equal scores by ID
		{"pasta -salt", []string{}},                    // excluded away: no hits is not an error
		{"rust", []string{}},                           // unknown term
		{"channels -go", []string{}},                   // every channels document mentions go
		{"generics constraints", []string{"d2"}},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			results, err := ix.Search(tt.query)
			if err != nil {
				t.Fatalf("Search(%q): %v", tt.query, err)
			}
			if got := ids(results); !slices.Equal(got, tt.want) {
				t.Errorf("Search(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestSearchScores(t *testing.T) {
	ix := newTestIndex(t)
	results, _ := ix.Search("channels")
	// d5: "channels" in the title (2); d1: "channels" in the body (1). "channel" in d5's body is
	// a different word - the default analyzer does no stemming.
	want := []Result{{ID: "d5", Title: "Go channels", Score: 2}, {ID: "d1", Title: "Concurrency in Go", Score: 1}}
	if !slices.Equal(results, want) {
		t.Errorf("results = %+v, want %+v", results, want)
	}
}

func TestEmptyQuery(t *testing.T) {
	ix := newTestIndex(t)
	for _, q := range []string{"", "   ", "the and of", "-go"} {
		if _, err := ix.Search(q); !errors.Is(err, ErrEmptyQuery) {
			t.Errorf("Search(%q) error = %v, want ErrEmptyQuery", q, err)
		}
	}
}

func TestAddErrors(t *testing.T) {
	ix := newTestIndex(t)

	err := ix.Add(Document{ID: "d1", Title: "again"})
	docErr, ok := errors.AsType[*DocError](err)
	if !ok || docErr.ID != "d1" || !errors.Is(err, ErrDuplicateID) {
		t.Errorf("duplicate: error = %v, want a *DocError for d1 wrapping ErrDuplicateID", err)
	}
	if err := ix.Add(Document{ID: "  "}); !errors.Is(err, ErrEmptyID) {
		t.Errorf("blank id: error = %v, want ErrEmptyID", err)
	}

	// AddAll reports every bad document but still adds the good ones.
	err = ix.AddAll([]Document{
		{ID: "d6", Body: "new"},
		{ID: "d2", Body: "duplicate of an existing one"},
		{ID: "", Body: "no id"},
		{ID: "d7", Body: "also new"},
		{ID: "d6", Body: "duplicate inside the batch"},
	}, 2)
	if !errors.Is(err, ErrDuplicateID) || !errors.Is(err, ErrEmptyID) {
		t.Errorf("AddAll error = %v, want both ErrDuplicateID and ErrEmptyID", err)
	}
	if n := strings.Count(err.Error(), "\n") + 1; n != 3 {
		t.Errorf("AddAll reported %d errors, want 3:\n%v", n, err)
	}
	if ix.Len() != len(corpus)+2 {
		t.Errorf("Len = %d, want %d", ix.Len(), len(corpus)+2)
	}
}

func TestTerms(t *testing.T) {
	ix := NewIndex(nil)
	_ = ix.Add(Document{ID: "a", Body: "beta alpha"})
	_ = ix.Add(Document{ID: "b", Body: "alpha gamma"})

	var got []string
	for term, docs := range ix.Terms() {
		got = append(got, fmt.Sprintf("%s:%d", term, docs))
	}
	if want := []string{"alpha:2", "beta:1", "gamma:1"}; !slices.Equal(got, want) {
		t.Errorf("Terms = %v, want %v", got, want)
	}
}

func TestSet(t *testing.T) {
	a, b := NewSet(1, 2, 3, 4), NewSet(3, 4, 5)
	inter := slices.Sorted(a.Intersect(b).All())
	diff := slices.Sorted(a.Difference(b).All())
	if !slices.Equal(inter, []int{3, 4}) || !slices.Equal(diff, []int{1, 2}) {
		t.Errorf("Intersect = %v, Difference = %v", inter, diff)
	}
	if a.Len() != 4 {
		t.Error("Intersect and Difference must not change the receiver")
	}
}

func TestSaveAndLoad(t *testing.T) {
	ix := newTestIndex(t)
	var buf bytes.Buffer
	if err := ix.Save(&buf); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(&buf, nil, 4)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	before, _ := ix.Search("go concurrency")
	after, _ := loaded.Search("go concurrency")
	if !slices.Equal(before, after) {
		t.Errorf("after a round trip: %v, before: %v", after, before)
	}

	if _, err := Load(strings.NewReader("{not json"), nil, 1); err == nil {
		t.Error("Load of invalid JSON returned no error")
	}
	dup := `[{"id":"x","title":"one"},{"id":"x","title":"two"}]`
	if _, err := Load(strings.NewReader(dup), nil, 1); !errors.Is(err, ErrDuplicateID) {
		t.Errorf("Load with duplicate ids: error = %v, want ErrDuplicateID", err)
	}
}

// TestConcurrentAddAndSearch adds documents while other goroutines search. It is a job for
// go test -race: the RWMutex must keep every access synchronised.
func TestConcurrentAddAndSearch(t *testing.T) {
	ix := newTestIndex(t)
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			_ = ix.Add(Document{ID: fmt.Sprintf("new-%d", i), Body: "go concurrency news"})
		})
		wg.Go(func() {
			if _, err := ix.Search("go concurrency"); err != nil {
				t.Errorf("Search: %v", err)
			}
			for range ix.Terms() {
				break
			}
		})
	}
	wg.Wait()
	if results, _ := ix.Search("news"); len(results) != 50 {
		t.Errorf("found %d new documents, want 50", len(results))
	}
}

func ExampleIndex_Search() {
	ix := NewIndex(nil) // nil: the default analyzer
	_ = ix.AddAll([]Document{
		{ID: "go-intro", Title: "Learning Go", Body: "Go is simple. Goroutines make concurrency easy."},
		{ID: "java-intro", Title: "Learning Java", Body: "Java has threads for concurrency."},
		{ID: "go-generics", Title: "Go generics", Body: "Type parameters in Go."},
	}, 2)

	for _, q := range []string{"go", "concurrency", "concurrency -java", "the"} {
		results, err := ix.Search(q)
		if err != nil {
			fmt.Printf("%-18q error: %v\n", q, err)
			continue
		}
		fmt.Printf("%-18q", q)
		for _, r := range results {
			fmt.Printf(" %s(%d)", r.ID, r.Score)
		}
		fmt.Println()
	}
	// Output:
	// "go"               go-generics(3) go-intro(3)
	// "concurrency"      go-intro(1) java-intro(1)
	// "concurrency -java" go-intro(1)
	// "the"              error: search "the": query has no search terms
}
