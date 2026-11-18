package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ============================================================================================
// 018 - HTTP services with net/http
// ============================================================================================
//
// What you build: a small JSON API for users, with nothing but the standard library.
//
//	1. a handler       the one interface the whole server side of net/http rests on
//	2. the users API   a router (ServeMux) sends each "METHOD /path" to its handler
//	3. bad requests    every mistake gets a JSON answer with the right status code
//	4. middleware      code that runs around every request: logging, panic recovery
//	5. a real server   timeouts, a client, and a graceful shutdown
//
// Each step is one function below, in this order, followed by the types it introduces.

// httpServices runs the steps of section 018.
func httpServices() {
	fmt.Println("--- 018 HTTP services with net/http ---")
	handlerBasics()    // 1. a handler
	usersAPI()         // 2. the users API
	badRequests()      // 3. bad requests
	middlewareBasics() // 4. middleware
	realServer()       // 5. a real server

	// More: basics/018_http_services
}

// --------------------------------------------------------------------------------------------
// Step 1 - a handler
// --------------------------------------------------------------------------------------------

// handlerBasics is step 1. Everything on the server side of net/http rests on one interface:
//
//	type Handler interface { ServeHTTP(w http.ResponseWriter, r *http.Request) }
//
// r is the request that came in, and w is where the answer goes.
func handlerBasics() {
	step("1. a handler: write the headers first, then the status, then the body")

	// http.HandlerFunc turns an ordinary function into a Handler.
	hello := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8") // 1. headers
		w.WriteHeader(http.StatusOK)                                // 2. the status, only once
		fmt.Fprintf(w, "hello from %s %s", r.Method, r.URL.Path)    // 3. the body
	})
	show("GET /hi -> %s", callAPI(hello, "GET", "/hi", ""))

	// Why the order matters: the first Write sends status 200 by itself, and a header set
	// after that is silently ignored.
}

// --------------------------------------------------------------------------------------------
// Step 2 - the users API
// --------------------------------------------------------------------------------------------

// usersAPI is step 2: the users API and its routes. The users are kept in memory here;
// 019 and 020 move them into PostgreSQL without changing a single handler.
func usersAPI() {
	step("2. the users API: a ServeMux sends each request to its handler")
	api := usersMux(NewMemoryUserRepo())
	show("POST   /users     -> %s", callAPI(api, "POST", "/users", `{"name":"Ada","email":"ada@example.com"}`))
	show("GET    /users/1   -> %s", callAPI(api, "GET", "/users/1", ""))
	// Each request runs in its own goroutine, so the storage behind the API must be safe for
	// concurrent use - that is the mutex in MemoryUserRepo.
}

// User is the resource the API serves. The json tags choose the names in the JSON bodies.
type User struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ErrDuplicate means "this email is already taken". The other error of the users API is
// ErrNotFound from section 009.
var ErrDuplicate = errors.New("already exists")

// UserRepository is the storage the handlers need. The handlers know only this interface, not
// where the users really live: here it is a map, in 019 and 020 a database. Section 019
// explains the idea - it is called the repository pattern.
type UserRepository interface {
	Create(ctx context.Context, u *User) error // fills in u.ID
	ByID(ctx context.Context, id int64) (User, error)
	List(ctx context.Context) ([]User, error)
	Delete(ctx context.Context, id int64) error
}

// MemoryUserRepo is a UserRepository that keeps users in a map. It needs no database, which
// also makes it a perfect stand-in in tests.
type MemoryUserRepo struct {
	mu     sync.RWMutex // handlers run concurrently (011)
	nextID int64
	users  map[int64]User
}

// The compiler checks that MemoryUserRepo satisfies UserRepository (008).
var _ UserRepository = (*MemoryUserRepo)(nil)

func NewMemoryUserRepo() *MemoryUserRepo { return &MemoryUserRepo{users: map[int64]User{}} }

func (r *MemoryUserRepo) Create(_ context.Context, u *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.users {
		if existing.Email == u.Email {
			return ErrDuplicate
		}
	}
	r.nextID++
	u.ID = r.nextID
	r.users[u.ID] = *u
	return nil
}

func (r *MemoryUserRepo) ByID(_ context.Context, id int64) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (r *MemoryUserRepo) List(_ context.Context) ([]User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]User, 0, len(r.users)) // not nil: an empty list is [] in JSON, not null
	for _, u := range r.users {
		out = append(out, u)
	}
	slices.SortFunc(out, func(a, b User) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}

func (r *MemoryUserRepo) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[id]; !ok {
		return ErrNotFound
	}
	delete(r.users, id)
	return nil
}

// usersMux builds the users API on the standard router, http.ServeMux. Since Go 1.22 a route
// is written "METHOD /path", and {id} is a wildcard whose value r.PathValue("id") returns.
// The helpers it uses (decodeJSON, writeJSON, writeError ...) are explained in step 3.
func usersMux(repo UserRepository) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {
		users, err := repo.List(r.Context()) // r.Context() ends when the client goes away
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, users)
	})
	mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
		var u User
		if err := decodeJSON(w, r, &u); err != nil {
			writeError(w, err)
			return // writing the error and FORGETTING to return is the classic handler bug
		}
		if err := validateUser(u); err != nil {
			writeError(w, err)
			return
		}
		if err := repo.Create(r.Context(), &u); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, u)
	})
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		u, err := repo.ByID(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, u)
	})
	mux.HandleFunc("DELETE /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		if err := repo.Delete(r.Context(), id); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent) // 204: success, and no body
	})
	return mux
}

// --------------------------------------------------------------------------------------------
// Step 3 - bad requests
// --------------------------------------------------------------------------------------------

// badRequests is step 3: every mistake gets a JSON answer with the right status code, and an
// unexpected error never shows its details to the client.
func badRequests() {
	step("3. bad requests: every error becomes JSON with the right status code")
	api := usersMux(NewMemoryUserRepo())
	_ = callAPI(api, "POST", "/users", `{"name":"Ada","email":"ada@example.com"}`) // Ada exists now
	for _, c := range []struct{ method, path, body string }{
		{"POST", "/users", `{"name":"Ada","email":"ada@example.com"}`}, // the same email again
		{"POST", "/users", `{"name":"","email":"x@example.com"}`},      // no name
		{"POST", "/users", `{"name":"Bob","emial":"bob@example.com"}`}, // a typo in a field name
		{"GET", "/users/42", ""},  // no such user
		{"GET", "/users/abc", ""}, // not a number
		{"DELETE", "/users", ""},  // /users has only GET and POST: 405, without writing any code
	} {
		show("%-6s %-10s -> %s", c.method, c.path, callAPI(api, c.method, c.path, c.body))
	}
}

// InputError is a mistake made by the client. It becomes a 400 with its message. Any other,
// unexpected error becomes a generic 500 whose details never leave the server.
type InputError struct{ Msg string }

func (e *InputError) Error() string { return e.Msg }

// statusFor turns an error into an HTTP status and a message that is safe to show. This is the
// one place where domain errors meet HTTP; errors.Is and errors.AsType (009) recognise them.
// The Gin version of the API (018b) uses it too.
func statusFor(err error) (int, string) {
	if inputErr, ok := errors.AsType[*InputError](err); ok {
		return http.StatusBadRequest, inputErr.Msg
	}
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, "not found"
	case errors.Is(err, ErrDuplicate):
		return http.StatusConflict, "already exists"
	case errors.Is(err, ErrInvalidCredentials): // the failed login of 018c
		return http.StatusUnauthorized, "invalid credentials"
	default:
		// A real service logs err here, with the request ID. The client sees nothing of it.
		return http.StatusInternalServerError, "internal error"
	}
}

// writeError is the one place where an error is turned into a response.
func writeError(w http.ResponseWriter, err error) {
	status, msg := statusFor(err)
	writeJSON(w, status, map[string]string{"error": msg})
}

// writeJSON is the one place where a body is encoded. The Content-Type goes BEFORE the status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) // the status is already sent: an error here can only be logged
}

// decodeJSON reads the request body into dst. MaxBytesReader caps the body - without it a
// client could stream gigabytes - and DisallowUnknownFields turns a typo into an error.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return &InputError{Msg: "invalid JSON: " + err.Error()}
	}
	return nil
}

// validateUser checks a new user by hand. Gin does the same with tags on a struct (018b).
func validateUser(u User) error {
	switch {
	case strings.TrimSpace(u.Name) == "":
		return &InputError{Msg: "name is required"}
	case !strings.Contains(u.Email, "@"):
		return &InputError{Msg: "email is invalid"}
	}
	return nil
}

// parseID turns the {id} from the path into a number.
func parseID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, &InputError{Msg: "id must be a number"}
	}
	return id, nil
}

// --------------------------------------------------------------------------------------------
// Step 4 - middleware
// --------------------------------------------------------------------------------------------

// middlewareBasics is step 4. A middleware is a function that takes a handler and returns a
// new handler wrapped around it. Its code runs before and after every request, which makes it
// the place for logging, panic recovery, authentication (018c) and the like.
func middlewareBasics() {
	step("4. middleware: code that runs around every request")
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("a bug") })
	safe := chain(boom, logRequests, recoverPanics)
	show("a panicking handler behind recoverPanics -> %s", callAPI(safe, "GET", "/boom", ""))
}

// middleware is the shape of every net/http middleware.
type middleware func(http.Handler) http.Handler

// chain wraps h in the middleware. The FIRST one listed is the outermost: it sees the request
// first and the response last. With logRequests first, it also logs the 500 that
// recoverPanics writes - the same order gin.Default() uses (Logger, then Recovery).
func chain(h http.Handler, mws ...middleware) http.Handler {
	for _, mw := range slices.Backward(mws) {
		h = mw(h)
	}
	return h
}

// logRequests prints one line per request. A real logger would use slog (013) and record the
// duration too; here the duration is left out, so the output is the same on every run.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK} // no WriteHeader call = 200
		next.ServeHTTP(rec, r)                                           // before ... next ... after
		show("log: %s %s -> %d", r.Method, r.URL.Path, rec.status)
	})
}

// statusRecorder remembers the status code, which a ResponseWriter cannot tell you. It EMBEDS
// the real http.ResponseWriter (007), so it is a ResponseWriter itself and overrides only
// WriteHeader. Unwrap lets http.ResponseController reach the original (for Flush and deadlines).
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// recoverPanics turns a panic in any handler into a clean 500. Without it, net/http survives
// the panic but closes the connection without sending any response at all.
func recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				writeError(w, fmt.Errorf("panic: %v", v)) // unexpected -> the generic 500
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// --------------------------------------------------------------------------------------------
// Step 5 - a real server
// --------------------------------------------------------------------------------------------

// realServer is step 5: the API on a real network port, called by a real HTTP client, and shut
// down cleanly at the end.
func realServer() {
	step("5. a real server: timeouts, a client, a graceful shutdown")

	// Port 0 asks the operating system for any free port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		show("cannot listen: %v", err)
		return
	}
	srv := newHTTPServer(chain(usersMux(NewMemoryUserRepo()), logRequests, recoverPanics))
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }() // Serve blocks, so it gets its own goroutine
	base := "http://" + ln.Addr().String()

	client := &http.Client{Timeout: 5 * time.Second} // the default client would wait forever
	show("client POST -> %s", send(client, "POST", base+"/users", `{"name":"Ann","email":"ann@example.com"}`))
	show("client GET  -> %s", send(client, "GET", base+"/users", ""))

	// Shutdown stops accepting connections, waits for the requests in flight, then returns.
	// After it, Serve returns http.ErrServerClosed - which means success, not failure.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		show("shutdown: %v", err)
	}
	show("Serve returned: %v", <-serveErr)

	// A real main() runs until Ctrl+C, or until the SIGTERM that Docker and Kubernetes send,
	// and then shuts down gracefully, so that no request in flight is cut off:
	//
	//	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	//	defer stop()
	//	go func() { serveErr <- srv.ListenAndServe() }()
	//	<-ctx.Done()                                                   // a signal arrived
	//	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//	defer cancel()
	//	err := srv.Shutdown(shutdownCtx)                               // drain, then close
	//	db.Close()                                                     // only after the drain
}

// newHTTPServer builds a server WITH timeouts. http.ListenAndServe(":8080", h) uses a server
// without any: a client that never finishes sending its headers keeps a connection forever.
func newHTTPServer(h http.Handler) *http.Server {
	return &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,  // the one you must never leave out
		ReadTimeout:       15 * time.Second, // the whole request, body included
		WriteTimeout:      30 * time.Second, // longer than the slowest handler
		IdleTimeout:       2 * time.Minute,  // keep-alive connections between requests
	}
}

// send makes one request over the network and returns "status body". The response body must
// ALWAYS be closed, or the connection can never be reused.
func send(client *http.Client, method, url, body string) string {
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return "error: " + err.Error()
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return "error: " + err.Error()
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "error: " + err.Error()
	}
	return fmt.Sprintf("%d %s", resp.StatusCode, strings.TrimSpace(string(data)))
}
