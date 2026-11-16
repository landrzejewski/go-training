// Command 018_http_services is module 018 of the Go course: HTTP services.
//
// net/http is a production-grade HTTP client and server: TLS, HTTP/2, connection pooling,
// graceful shutdown and (since Go 1.22) method- and wildcard-aware routing, with no framework.
// Everything runs offline: the demo servers bind an ephemeral loopback port and stop again.
//
//	go run ./basics/018_http_services
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

func main() {
	handlers()          // Handler, HandlerFunc, header/status/body order, MaxBytesReader
	routing()           // Go 1.22 ServeMux patterns, 405, precedence, conflicts
	middleware()        // func(http.Handler) http.Handler, chaining, recovery, status recorder
	errorHandling()     // one error type, one writer, handlers that return error
	serverAndTimeouts() // an explicit http.Server, timeouts, net.Listen on :0
	gracefulShutdown()  // signal.NotifyContext, srv.Shutdown drains in-flight requests
	testingHandlers()   // httptest.NewRecorder, NewServer, a fake RoundTripper
}

// handlers shows the Handler interface and how a response must be written.
func handlers() {
	fmt.Println("--- handlers ---")

	// The server side rests on one interface: Handler { ServeHTTP(ResponseWriter, *Request) }.
	// A handler can be a type with state...
	var counter atomic.Int64
	structHandler := &countingHandler{count: &counter}

	// ...or a function: http.HandlerFunc is a defined function type WITH a ServeHTTP method
	// (module 007), so a closure satisfies the interface.
	// ResponseWriter has Header(), WriteHeader(status), Write(body) - and the ORDER matters.
	funcHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8") // 1. headers first
		w.WriteHeader(http.StatusTeapot)                            // 2. status, once
		fmt.Fprintf(w, "method=%s path=%s", r.Method, r.URL.Path)   // 3. then the body
	})

	// httptest.NewRecorder runs a handler with no server at all (see testingHandlers).
	rec := httptest.NewRecorder()
	structHandler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/a", nil))
	structHandler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/b", nil))
	fmt.Println("a Handler with state, after 2 requests: count =", counter.Load())

	rec = httptest.NewRecorder()
	funcHandler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/teapot", nil))
	fmt.Printf("a HandlerFunc: status=%d contentType=%q body=%q\n",
		rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())

	// The most common beginner bug: the wrong order.
	wrong := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("body first"))                 // the first Write implies WriteHeader(200)
		w.Header().Set("X-Too-Late", "ignored")       // set after the body: never sent
		w.WriteHeader(http.StatusInternalServerError) // logs "superfluous WriteHeader", not applied
	})
	rec = httptest.NewRecorder()
	wrong.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	// rec.Header() is the LIVE map; rec.Result() is the snapshot a client would actually see.
	fmt.Printf("wrong order: X-Too-Late=%q, status stayed %d\n",
		rec.Result().Header.Get("X-Too-Late"), rec.Code)

	// ALWAYS cap the request body, or a client can stream gigabytes into your io.ReadAll.
	// (r.Body is an io.ReadCloser the server closes for you.)
	limited := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 16) // before reading a single byte
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		fmt.Fprintf(w, "read %d bytes", len(data))
	})
	for _, body := range []string{"small", strings.Repeat("x", 100)} {
		rec = httptest.NewRecorder()
		limited.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
		fmt.Printf("MaxBytesReader, %3d-byte body: status=%d %q\n",
			len(body), rec.Code, strings.TrimSpace(rec.Body.String()))
	}

	// r.Context() is cancelled when the client disconnects - pass it to every downstream call.
	// Each request runs in its own goroutine, so shared state needs a mutex or an atomic.
}

type countingHandler struct{ count *atomic.Int64 }

func (h *countingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.count.Add(1) // atomic: requests run concurrently
	fmt.Fprintln(w, "ok")
}

// routing shows the Go 1.22 ServeMux pattern language: [METHOD ]/path with wildcards.
func routing() {
	fmt.Println("\n--- routing with ServeMux ---")

	// Before 1.22 ServeMux matched only paths, so everyone imported gorilla/mux, chi or gin.
	// A pattern is [METHOD ]/path; r.PathValue reads a wildcard (r.SetPathValue is for tests).
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "list of items") // GET only
	})
	mux.HandleFunc("POST /items", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "created an item")
	})
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "item %s", r.PathValue("id")) // a wildcard segment
	})
	mux.HandleFunc("GET /items/{id}/tags/{tag}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "item %s tag %s", r.PathValue("id"), r.PathValue("tag"))
	})
	mux.HandleFunc("GET /files/{path...}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "file at %q", r.PathValue("path")) // a trailing wildcard: the rest
	})
	mux.HandleFunc("GET /exact/{$}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "matched /exact/ exactly") // {$} anchors the match
	})
	// A pattern without a method matches all methods. A trailing slash ("/static/") still
	// means a whole subtree - {$} is how you match only the exact path.

	// HandleFunc wants a FUNCTION; a type implementing http.Handler goes to mux.Handle:
	//	mux.HandleFunc("GET /x", countingHandler{}) // ERROR: cannot use countingHandler{} (value of struct type countingHandler) as func(http.ResponseWriter, *http.Request) value in argument to mux.HandleFunc
	// and the signature is fixed - both parameters, in that order:
	//	mux.HandleFunc("GET /y", func(w http.ResponseWriter) {}) // ERROR: cannot use func(w http.ResponseWriter) {} (value of type func(w http.ResponseWriter)) as func(http.ResponseWriter, *http.Request) value in argument to mux.HandleFunc

	for _, probe := range []struct{ method, path string }{
		{http.MethodGet, "/items"},
		{http.MethodPost, "/items"},
		{http.MethodDelete, "/items"}, // registered for GET/POST only -> 405 + Allow, for free
		{http.MethodGet, "/items/42"},
		{http.MethodGet, "/items/42/tags/urgent"},
		{http.MethodGet, "/files/a/b/c.txt"},
		{http.MethodGet, "/exact/"},
		{http.MethodGet, "/exact/deeper"},
		{http.MethodGet, "/nothing-here"},
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(probe.method, probe.path, nil))
		extra := ""
		if allow := rec.Header().Get("Allow"); allow != "" {
			extra = "  Allow: " + allow
		}
		fmt.Printf("%-6s %-22s -> %d %s%s\n",
			probe.method, probe.path, rec.Code, strings.TrimSpace(rec.Body.String()), extra)
	}

	// Precedence is by SPECIFICITY, not registration order: a literal GET /items/42 would beat
	// GET /items/{id}. Two patterns where neither is more specific PANIC at registration -
	// a routing conflict is a startup crash, never a silent mis-route.
	conflict := fmt.Sprint(catchPanic(func() {
		bad := http.NewServeMux()
		bad.HandleFunc("GET /a/{x}/c", func(http.ResponseWriter, *http.Request) {})
		bad.HandleFunc("GET /a/b/{y}", func(http.ResponseWriter, *http.Request) {})
	}))
	fmt.Println("registering `GET /a/{x}/c` and `GET /a/b/{y}` panics:")
	for _, line := range strings.Split(conflict, "\n")[1:3] { // line 0 holds source paths
		fmt.Println("  " + line)
	}

	// http.StripPrefix + http.FileServer + http.FS (over an embed.FS, module 015) serve static
	// files in three lines. What a router/framework still buys: route groups with shared
	// middleware, binding and validation. examples/rest.go uses Gin, which predates 1.22; for a
	// new service start with net/http and add a router when you can name what it gives you.
}

// catchPanic runs f and returns whatever it panicked with (a local copy of module 005's helper).
func catchPanic(f func()) (recovered any) {
	defer func() {
		if r := recover(); r != nil {
			recovered = r
		}
	}()
	f()
	return "did not panic"
}

// middleware shows wrapping handlers: a handler that takes a handler is just a function.
func middleware() {
	fmt.Println("\n--- middleware ---")

	log := quietLogger()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ok", func(w http.ResponseWriter, r *http.Request) {
		id, _ := r.Context().Value(requestIDKey).(string)
		fmt.Fprintf(w, "handled %s", id)
	})
	mux.HandleFunc("GET /boom", func(w http.ResponseWriter, r *http.Request) {
		panic("something went badly wrong")
	})

	// Order reads OUTERMOST first: the first listed sees the request first and the response
	// last. Recovery goes outermost so a panic anywhere inside is caught.
	handler := chain(mux, recovery(log), requestID, logging(log))

	for _, path := range []string{"/ok", "/ok", "/boom"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		fmt.Printf("GET %-5s -> %d X-Request-Id=%s body=%s\n", path, rec.Code,
			rec.Header().Get("X-Request-Id"), strings.TrimSpace(rec.Body.String()))
	}
	// net/http itself recovers a handler panic per connection (one bad request cannot take the
	// server down), but it logs a stack trace and closes the connection with NO response.
	// recovery turns it into a clean 500 JSON body, logged like everything else.

	// net/http has no route groups: for middleware on a subset of routes, wrap a second
	// ServeMux or the individual handlers.
}

// middlewareFunc is the shape every middleware takes.
type middlewareFunc func(http.Handler) http.Handler

// chain applies middleware so that the FIRST listed is the outermost.
func chain(h http.Handler, mw ...middlewareFunc) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

// Request-scoped values go in the context under an unexported key type (module 011).
type ctxKey string

const requestIDKey ctxKey = "requestID"

func requestID(next http.Handler) http.Handler {
	var seq atomic.Int64
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := fmt.Sprintf("req-%03d", seq.Add(1))
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}

func logging(log *slog.Logger) middlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r) // before ... next ... after
			id, _ := r.Context().Value(requestIDKey).(string)
			log.Info("request", slog.String("id", id), slog.String("method", r.Method),
				slog.String("path", r.URL.Path), slog.Int("status", rw.status),
				slog.Int("bytes", rw.written), slog.Bool("fast", time.Since(start) < time.Second))
		})
	}
}

func recovery(log *slog.Logger) middlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered", slog.Any("value", rec), slog.String("path", r.URL.Path))
					writeError(w, r, errors.New("panic")) // an unexpected error -> generic 500
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// statusRecorder captures the status and byte count - ResponseWriter has no getter for them.
// A naive wrapper HIDES http.Flusher, http.Hijacker and io.ReaderFrom (breaking streaming,
// WebSockets and io.Copy fast paths): implement what you need, or provide Unwrap so
// http.ResponseController (Go 1.20) can reach Flush and SetWriteDeadline through the wrapper.
type statusRecorder struct {
	http.ResponseWriter
	status  int
	written int
	wrote   bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.wrote {
		return // a second WriteHeader is a no-op, as in the real ResponseWriter
	}
	r.status, r.wrote = status, true
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wrote {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(b)
	r.written += n
	return n, err
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// quietLogger writes to stdout without timestamps, so the course output stays reproducible.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
}

// errorHandling shows one error shape for the whole API and handlers that return error.
func errorHandling() {
	fmt.Println("\n--- consistent error handling ---")

	// http.Handler returns nothing, so the compiler cannot make you handle errors - and handlers
	// drift: plain text here, JSON there, a forgotten status, a leaked internal message.
	// Fix 1: ONE error type and ONE writer. Fix 2: handlers that return error (apiHandler), so
	// a handler can `return err` - no more writing an error and falling through to success.
	mux := http.NewServeMux()
	mux.Handle("GET /items/{id}", apiHandler(func(w http.ResponseWriter, r *http.Request) error {
		switch id := r.PathValue("id"); id {
		case "1":
			writeJSON(w, http.StatusOK, map[string]any{"id": 1, "name": "first"})
			return nil
		case "missing":
			return mapDomainError(fmt.Errorf("lookup %q: %w", id, errNotFound))
		case "secret":
			return mapDomainError(fmt.Errorf("lookup %q: %w", id, errPermission))
		case "slow":
			return mapDomainError(fmt.Errorf("lookup %q: %w", id, context.DeadlineExceeded))
		default:
			// An unexpected failure whose message the client must NEVER see.
			return errors.New("database connection string is postgres://admin:hunter2@db/prod")
		}
	}))

	handler := chain(mux, requestID)
	for _, id := range []string{"1", "missing", "secret", "slow", "kaboom"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/items/"+id, nil))
		fmt.Printf("GET /items/%-8s -> %d %s\n", id, rec.Code, strings.TrimSpace(rec.Body.String()))
	}
	// The last one did not leak the connection string. Use http.Error only in plain-text
	// services: a JSON API answers JSON on errors too, or every client needs two parsers.
	// (http.Error also sets X-Content-Type-Options: nosniff for you.)
}

// The domain's sentinel errors (module 009).
var (
	errNotFound   = errors.New("not found")
	errPermission = errors.New("permission denied")
)

// apiError is the single error shape this service ever returns: a status, a stable machine
// code and a human detail.
type apiError struct {
	Status int    `json:"-"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
	ReqID  string `json:"requestId,omitempty"`
}

func (e apiError) Error() string { return e.Code + ": " + e.Detail }

// writeJSON is the one place a body is serialised. Content-Type goes before the status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// The status is already sent and cannot change - but a half-written body is worth a log.
		slog.Warn("write json body", slog.Any("err", err))
	}
}

// writeError is the one place a failure is serialised. Everything funnels through it.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var apiErr apiError
	if !errors.As(err, &apiErr) {
		// Unexpected: a generic 500. Log the real error with the request ID, never send it.
		apiErr = apiError{Status: http.StatusInternalServerError, Code: "internal",
			Detail: "the server encountered an unexpected condition"}
	}
	if id, ok := r.Context().Value(requestIDKey).(string); ok {
		apiErr.ReqID = id
	}
	writeJSON(w, apiErr.Status, apiErr)
}

// apiHandler lets a handler return an error instead of remembering to write AND return.
type apiHandler func(http.ResponseWriter, *http.Request) error

func (h apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		writeError(w, r, err)
	}
}

// mapDomainError turns sentinels into API errors - at the boundary and nowhere else.
func mapDomainError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, errNotFound):
		return apiError{Status: http.StatusNotFound, Code: "not_found", Detail: "no such item"}
	case errors.Is(err, errPermission):
		return apiError{Status: http.StatusForbidden, Code: "forbidden", Detail: "not allowed"}
	case errors.Is(err, context.DeadlineExceeded):
		return apiError{Status: http.StatusGatewayTimeout, Code: "timeout", Detail: "upstream timed out"}
	default:
		return err // unexpected: writeError turns it into a generic 500
	}
}

// serverAndTimeouts shows a properly configured http.Server serving real requests.
func serverAndTimeouts() {
	fmt.Println("\n--- http.Server and timeouts ---")

	before := runtime.NumGoroutine()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello/{name}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"hello": r.PathValue("name")})
	})

	// net.Listen on port 0 asks the OS for a free port and ln.Addr() says which - what tests
	// should do instead of hard-coding one. Loopback also avoids the macOS firewall prompt.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("could not listen:", err)
		return
	}
	srv := newServer(mux)
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	base := "http://" + ln.Addr().String()
	fmt.Println("listening on", base)

	client := &http.Client{Timeout: 5 * time.Second} // the client needs a timeout too
	fmt.Println("GET /hello/gopher ->", fetch(client, base+"/hello/gopher"))
	fmt.Println("GET /nothing      ->", fetch(client, base+"/nothing"))

	// Serve (and ListenAndServe) return http.ErrServerClosed after Shutdown: a SUCCESS.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	fmt.Println("Serve returned:", <-serveErr, "- a clean shutdown, not a failure")

	time.Sleep(50 * time.Millisecond) // let connection goroutines unwind before counting
	fmt.Printf("goroutines before=%d after=%d (no leak)\n", before, runtime.NumGoroutine())

	// TLS: ListenAndServeTLS (+ golang.org/x/crypto/acme/autocert for Let's Encrypt);
	// HTTP/2 is enabled automatically over TLS.
}

// fetch GETs url and returns "status body". Always close resp.Body, or the connection leaks.
func fetch(client *http.Client, url string) string {
	resp, err := client.Get(url)
	if err != nil {
		return "error: " + err.Error()
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return fmt.Sprintf("%d %s", resp.StatusCode, strings.TrimSpace(string(body)))
}

// newServer builds a properly configured server. http.ListenAndServe(addr, h) uses a zero
// http.Server, which has NO timeouts: a client that connects and never sends a byte holds a
// goroutine and a file descriptor forever (Slowloris). Fine for demos, never for production.
func newServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,   // the one you must not omit: bounds a silent client
		ReadTimeout:       15 * time.Second,  // the whole request, headers plus body
		WriteTimeout:      30 * time.Second,  // end of headers to end of response (> slowest handler)
		IdleTimeout:       120 * time.Second, // keep-alive connections between requests
		MaxHeaderBytes:    1 << 20,
		// ErrorLog: slog.NewLogLogger(handler, slog.LevelError) routes server errors to slog.
		// BaseContext/ConnContext attach values or cancellation to every request's context.
		// For streaming endpoints drop WriteTimeout, use ResponseController.SetWriteDeadline.
	}
}

// gracefulShutdown shows catching SIGINT/SIGTERM and draining in-flight requests.
func gracefulShutdown() {
	fmt.Println("\n--- graceful shutdown and OS signals ---")

	// Kubernetes sends SIGTERM, waits (30s by default), then SIGKILL - which cannot be caught.
	// Ignore SIGTERM and every in-flight request is dropped: 502s on each rolling deploy.
	before := runtime.NumGoroutine()
	var completed atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("GET /slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond) // still in flight when the signal arrives
		completed.Add(1)
		fmt.Fprint(w, "finished")
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("could not listen:", err)
		return
	}
	srv := newServer(mux)
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	base := "http://" + ln.Addr().String()

	// signal.NotifyContext (Go 1.16) returns a context cancelled on the first matching signal.
	// Real services pass os.Interrupt (SIGINT, Ctrl+C, portable) AND syscall.SIGTERM.
	// Register EARLY: a signal arriving before Notify gets the default action - terminate.
	// That is also what makes the self-signal below safe.
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop() // restores the default, so a SECOND Ctrl+C can still kill a wedged shutdown

	inFlight := make(chan string, 1)
	go func() { inFlight <- fetch(&http.Client{Timeout: 5 * time.Second}, base+"/slow") }()
	time.Sleep(50 * time.Millisecond) // let the request reach the handler

	// Send ourselves the signal, exactly as an orchestrator would.
	sent := false
	if p, err := os.FindProcess(os.Getpid()); err == nil {
		sent = p.Signal(os.Interrupt) == nil
	}
	if !sent {
		fmt.Println("(this platform cannot signal itself - cancelling the context instead)")
		stop() // Windows
	}
	select {
	case <-sigCtx.Done():
		fmt.Println("signal received: the context is cancelled, shutdown begins")
	case <-time.After(2 * time.Second):
		fmt.Println("no signal arrived within 2s - continuing anyway")
	}

	// Shutdown stops accepting, closes idle keep-alive connections and WAITS for in-flight
	// handlers - until they finish or its context expires. Give it less time than the
	// orchestrator's grace period, or SIGKILL arrives mid-drain. srv.Close() is the ungraceful
	// last resort. Shutdown does NOT cancel handler contexts: a handler blocked forever blocks
	// the shutdown until the timeout - pass a cancellable context via BaseContext if needed.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	shutdownErr := srv.Shutdown(ctx)
	fmt.Printf("Shutdown returned %v after %v\n", shutdownErr, time.Since(start).Round(10*time.Millisecond))
	fmt.Printf("the in-flight request still completed: %q, handlersFinished=%d\n",
		<-inFlight, completed.Load())
	fmt.Println("Serve returned:", <-serveErr)

	// Release in REVERSE order of acquisition, only after the drain (a running handler still
	// needs its database): 1. srv.Shutdown 2. db.Close 3. flush telemetry, close logs 4. return.
	// errgroup (golang.org/x/sync) runs the server and the signal watcher together.

	stop() // idempotent, so the deferred stop() stays correct
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		t.CloseIdleConnections()
	}
	time.Sleep(50 * time.Millisecond)
	// The extra goroutine is os/signal.loop, started by the process's FIRST signal.Notify and
	// never exiting by design - not a leak, but worth knowing before asserting NumGoroutine.
	fmt.Printf("goroutines before=%d after=%d (+1 is os/signal's permanent loop)\n",
		before, runtime.NumGoroutine())
}

// testingHandlers shows the three httptest tools: a recorder, a real server and a fake transport.
func testingHandlers() {
	fmt.Println("\n--- testing HTTP handlers ---")

	handler := apiHandler(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apiError{Status: http.StatusBadRequest, Code: "bad_request", Detail: "id required"}
		}
		writeJSON(w, http.StatusOK, map[string]string{"id": id})
		return nil
	})

	// 1. NewRequest + NewRecorder: no network, no server, no mux - the fastest handler test.
	// NewRequest panics on a malformed target: a test's own input should not be in doubt.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/items/42", nil)
	req.SetPathValue("id", "42") // outside a mux, PathValue is "" unless you set it
	handler.ServeHTTP(rec, req)
	// Assert on status, headers and the DECODED body - raw JSON strings break on field order.
	var decoded map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &decoded)
	fmt.Printf("NewRecorder: status=%d contentType=%q decoded=%v\n",
		rec.Code, rec.Header().Get("Content-Type"), decoded)

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/items/", nil))
	fmt.Printf("without SetPathValue: status=%d body=%s\n", rec.Code, strings.TrimSpace(rec.Body.String()))

	// 2. NewServer: a real server on a loopback port, for testing a CLIENT or real connections.
	// NewTLSServer adds a self-signed cert; srv.Client() trusts it. Go 1.27's
	// httptest.NewTestServer(t, h) registers its own t.Cleanup but returns an UNSTARTED
	// server - call srv.Start(). See TestHTTPTestServer in basics/014_testing/main_test.go.
	mux := http.NewServeMux()
	mux.Handle("GET /items/{id}", handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	fmt.Println("httptest.NewServer ->", fetch(srv.Client(), srv.URL+"/items/99"))

	// 3. http.RoundTripper is the client-side seam: a fake transport replaces a whole remote API.
	fake := &http.Client{Transport: fakeTransport(func(r *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"source":"fake transport"}`)),
		}
	})}
	fmt.Println("a fake RoundTripper answered:", fetch(fake, "https://api.example.com/anything"))
}

// fakeTransport adapts a function to http.RoundTripper.
type fakeTransport func(*http.Request) *http.Response

func (f fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r), nil }
