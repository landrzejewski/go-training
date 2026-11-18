package solutions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// --- 18.1 Greetings over HTTP ---

// NewGreetMux serves GET /hello/{name}. The optional query parameter lang picks the language:
// "pl", "de" or anything else for English. Other methods get 405 from the ServeMux itself.
func NewGreetMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello/{name}", func(w http.ResponseWriter, r *http.Request) {
		text := greetingFor(r.URL.Query().Get("lang"), r.PathValue("name"))
		writeJSON(w, http.StatusOK, map[string]string{"greeting": text})
	})
	return mux
}

func greetingFor(lang, name string) string {
	switch lang {
	case "pl":
		return "cześć, " + name
	case "de":
		return "hallo, " + name
	default:
		return "hello, " + name
	}
}

// writeJSON sets the Content-Type, then the status, then encodes the body - in that order.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// --- 18.2 A todo API ---

// Todo is one entry of the todo list.
type Todo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// ErrTodoNotFound is returned by TodoStore.Get for an unknown id.
var ErrTodoNotFound = errors.New("todo not found")

// TodoStore keeps todos in memory. It is safe for concurrent use: every request runs in its
// own goroutine.
type TodoStore struct {
	mu     sync.Mutex
	nextID int
	todos  map[int]Todo
}

func NewTodoStore() *TodoStore { return &TodoStore{todos: map[int]Todo{}} }

// Add stores a new todo with the next id, starting at 1.
func (s *TodoStore) Add(title string) Todo {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	t := Todo{ID: s.nextID, Title: title}
	s.todos[t.ID] = t
	return t
}

// Get returns the todo with the given id, or ErrTodoNotFound.
func (s *TodoStore) Get(id int) (Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.todos[id]
	if !ok {
		return Todo{}, ErrTodoNotFound
	}
	return t, nil
}

// All returns every todo sorted by id - an empty slice, never nil, so the JSON is [].
func (s *TodoStore) All() []Todo {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Todo, 0, len(s.todos))
	for _, t := range s.todos {
		out = append(out, t)
	}
	slices.SortFunc(out, func(a, b Todo) int { return a.ID - b.ID })
	return out
}

// maxTodoBody is the largest request body NewTodoMux accepts.
const maxTodoBody = 1 << 10 // 1 KB

// NewTodoMux serves the todo API:
//
//	GET  /todos       200 and the list
//	POST /todos       201 and the new todo; body {"title": "..."}
//	GET  /todos/{id}  200, 400 for an id that is not a number, 404 for an unknown id
//
// Errors are JSON too: {"error": "..."}.
func NewTodoMux(store *TodoStore) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /todos", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, store.All())
	})
	mux.HandleFunc("POST /todos", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxTodoBody)
		var in struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
				writeJSONError(w, http.StatusRequestEntityTooLarge, "body too large")
				return
			}
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		title := strings.TrimSpace(in.Title)
		if title == "" {
			writeJSONError(w, http.StatusBadRequest, "title is required")
			return
		}
		writeJSON(w, http.StatusCreated, store.Add(title))
	})
	mux.HandleFunc("GET /todos/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		t, err := store.Get(id)
		if errors.Is(err, ErrTodoNotFound) {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, t)
	})
	return mux
}

// --- 18.3 Middleware ---

// Middleware wraps a handler with code that runs before and after it.
type Middleware func(http.Handler) http.Handler

// Chain wraps h in mws. The FIRST middleware listed is the outermost: it sees the request first
// and the response last.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// RequireAPIKey lets a request through only when its X-API-Key header equals key. Otherwise it
// answers 401 {"error":"unauthorized"} and the wrapped handler never runs.
func RequireAPIKey(key string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-API-Key") != key {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// LogRequests writes one line per request to out: "METHOD PATH STATUS".
func LogRequests(out io.Writer) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			fmt.Fprintf(out, "%s %s %d\n", r.Method, r.URL.Path, rec.status)
		})
	}
}

// statusRecorder embeds the real ResponseWriter and remembers the status written through it.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}
