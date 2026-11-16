// Command 009_errors is module 009 of the Go course: errors.
//
// Go has no exceptions. An error is an ordinary value of the interface type
//
//	type error interface{ Error() string }
//
// returned next to the result and checked with an if. The control flow stays on the page; the
// price is `if err != nil`. panic is for programmer bugs, not expected failures (module 005).
//
//	go run ./basics/009_errors
package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

func main() {
	errorsAreValues() // the error interface, conventions, message style
	sentinelErrors()  // ErrXxx values, errors.Is, sentinels as API
	wrapping()        // %w vs %v, Unwrap, several %w, errors.Join, custom Is
	customTypes()     // payload-carrying types, errors.As, errors.AsType (Go 1.26)
	inPractice()      // handle once, partial results, deferred Close errors
}

// errorsAreValues shows that an error is just a value with an Error() string method.
func errorsAreValues() {
	fmt.Println("--- errors are values ---")

	// The error comes LAST; when it is non-nil the other results are not meaningful (documented
	// exceptions exist: io.Reader may return n > 0 together with an error).
	if v, err := parsePort("8080"); err == nil {
		fmt.Println("  parsePort(\"8080\") =", v)
	}
	for _, s := range []string{"http", "99999"} {
		if _, err := parsePort(s); err != nil {
			fmt.Printf("  parsePort(%q) -> %v\n", s, err)
		}
	}

	// errors.New and fmt.Errorf give an opaque value - its only capability is the message.
	// Anything with Error() string is an error: a struct...
	var err error = validationError{Field: "age", Value: -1}
	fmt.Printf("  a struct as an error: %v (%T)\n", err, err)
	// ...or a defined string type, which can even be a CONSTANT (errors.New is a function call).
	fmt.Printf("  a constant error: %v (%T)\n", errClosed, errClosed)

	// errors.New returns a new pointer every time, so equal text is not an equal error - that
	// is what makes comparing against a sentinel meaningful.
	a, b := errors.New("same text"), errors.New("same text")
	fmt.Println("  errors.New(\"x\") == errors.New(\"x\"):", a == b)

	// Message style: lower case, no trailing punctuation, no "error:"/"failed to" on the inner
	// message, and include the VALUES - messages get wrapped and concatenated:
	//	load config: open /etc/app.conf: no such file or directory
	fmt.Println("  good:", fmt.Errorf("parse port %q: value out of range", "99999"))
	fmt.Println("  bad:  \"Error: Failed to parse the port!\"")

	// Never drop an error silently. `_ = f()` documents a decision; errcheck and many CI setups
	// reject an unchecked call.
}

func parsePort(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parse port %q: %w", s, err) // %w keeps the strconv error reachable
	}
	if n < 1 || n > 65535 {
		return 0, fmt.Errorf("parse port %q: out of range 1..65535", s)
	}
	return n, nil
}

type validationError struct {
	Field string
	Value any
}

func (e validationError) Error() string { return fmt.Sprintf("invalid %s: %v", e.Field, e.Value) }

type constError string

func (e constError) Error() string { return string(e) }

const errClosed = constError("connection closed")

// sentinelErrors shows package-level error values that callers compare against.
func sentinelErrors() {
	fmt.Println("\n--- sentinel errors ---")

	// A sentinel is a package-level var named ErrXxx, made with errors.New, that callers branch
	// on: io.EOF, sql.ErrNoRows, os.ErrNotExist, context.Canceled. Export it only if callers
	// need it.
	s := &store{data: map[string]string{"a": "1"}}
	for _, key := range []string{"a", "missing"} {
		v, err := s.Get(key)
		switch {
		case err == nil:
			fmt.Printf("  Get(%q) = %q\n", key, v)
		case errors.Is(err, errNotFound):
			fmt.Printf("  Get(%q): not found, using a default (full error: %v)\n", key, err)
		default:
			fmt.Printf("  Get(%q): unexpected: %v\n", key, err)
		}
	}
	for _, key := range []string{"b", "a", "_private"} {
		err := s.Put(key, "x")
		fmt.Printf("  Put(%q): %v [exists=%t permission=%t]\n", key, err,
			errors.Is(err, errAlreadyExist), errors.Is(err, errPermission))
	}

	// Compare with errors.Is, not ==: == only matches an UNWRAPPED error, and almost every
	// real error gets wrapped somewhere.
	_, wrapped := s.Get("missing")
	fmt.Println("  wrapped == errNotFound:         ", wrapped == errNotFound, "<- wrapping broke it")
	fmt.Println("  errors.Is(wrapped, errNotFound):", errors.Is(wrapped, errNotFound))

	_, err := os.Open("/definitely/not/here")
	fmt.Println("  os.Open on a missing path: errors.Is(err, fs.ErrNotExist) =", errors.Is(err, fs.ErrNotExist))

	// A sentinel is part of your API - once callers branch on it you cannot remove it. Prefer,
	// in order: no sentinel (the caller just reports it); a behaviour to test for, like
	// net.Error's Timeout() bool; a custom type + errors.As when the caller needs details;
	// a sentinel for a simple fact with no payload. For "not supported here" wrap the standard
	// errors.ErrUnsupported (Go 1.21) instead of inventing one.
}

var (
	errNotFound     = errors.New("not found")
	errPermission   = errors.New("permission denied")
	errAlreadyExist = errors.New("already exists")
)

type store struct{ data map[string]string }

func (s *store) Get(key string) (string, error) {
	v, ok := s.data[key]
	if !ok {
		return "", fmt.Errorf("store get %q: %w", key, errNotFound) // context AND the sentinel
	}
	return v, nil
}

func (s *store) Put(key, value string) error {
	if _, exists := s.data[key]; exists {
		return fmt.Errorf("store put %q: %w", key, errAlreadyExist)
	}
	if strings.HasPrefix(key, "_") {
		return fmt.Errorf("store put %q: %w", key, errPermission)
	}
	s.data[key] = value
	return nil
}

// wrapping shows how errors gain context while keeping the cause reachable.
func wrapping() {
	fmt.Println("\n--- wrapping: %w, Unwrap, Is and Join ---")

	// Add context at each layer, saying what THAT layer was doing; the result reads as a path.
	// Do not repeat what the caller already knows.
	err := loadConfig("/etc/nonexistent.conf")
	fmt.Println(" ", err)

	// errors.Unwrap goes one step down the chain (rarely called directly); errors.Is searches
	// the whole chain for a value, errors.As for a type (see customTypes).
	for e := err; e != nil; e = errors.Unwrap(e) {
		fmt.Printf("    %-20T %v\n", e, e)
	}
	fmt.Println("  errors.Is(err, fs.ErrNotExist) =", errors.Is(err, fs.ErrNotExist))

	// %w wraps; %v only formats the cause into the text and THROWS IT AWAY. Wrapping is an API
	// decision: %w lets callers depend on the cause, %v hides an implementation detail.
	inner := errors.New("the cause")
	wrapped := fmt.Errorf("context: %w", inner)
	flattened := fmt.Errorf("context: %v", inner)
	fmt.Printf("  %%w: %v -> Is finds the cause: %t\n", wrapped, errors.Is(wrapped, inner))
	fmt.Printf("  %%v: %v -> Is finds the cause: %t\n", flattened, errors.Is(flattened, inner))

	// Go 1.20: several %w verbs wrap a tree of causes.
	multi := fmt.Errorf("two causes: %w and %w", errNotFound, errPermission)
	fmt.Printf("  %v: Is(notFound)=%t Is(permission)=%t\n", multi,
		errors.Is(multi, errNotFound), errors.Is(multi, errPermission))

	// Go 1.20 errors.Join: one error per line, searched by Is/As. Nil arguments are dropped and
	// joining nothing gives nil - ideal for collecting every validation failure.
	if err := validateUser("", -5, "not-an-email"); err != nil {
		for line := range strings.Lines(err.Error()) {
			fmt.Print("    ", line)
		}
		fmt.Println("\n    Is(errInvalidAge) =", errors.Is(err, errInvalidAge))
	}
	fmt.Println("  errors.Join(nil, nil) == nil:", errors.Join(nil, nil) == nil)

	// A type controls unwrapping with Unwrap() error or Unwrap() []error, and matching with
	// Is(error) bool - when several values should count as the same condition.
	fmt.Println("  httpError{404} matches errNotFound via its Is method:", errors.Is(httpError{Code: 404}, errNotFound))
}

func loadConfig(path string) error {
	if err := readFile(path); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return nil
}

func readFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return f.Close()
}

var (
	errInvalidName  = errors.New("name must not be empty")
	errInvalidAge   = errors.New("age must be non-negative")
	errInvalidEmail = errors.New("email must contain @")
)

// validateUser reports EVERY problem at once rather than only the first.
func validateUser(name string, age int, email string) error {
	var problems []error
	if name == "" {
		problems = append(problems, errInvalidName)
	}
	if age < 0 {
		problems = append(problems, fmt.Errorf("%w: got %d", errInvalidAge, age))
	}
	if !strings.Contains(email, "@") {
		problems = append(problems, fmt.Errorf("%w: got %q", errInvalidEmail, email))
	}
	return errors.Join(problems...) // nil when problems is empty
}

type httpError struct{ Code int }

func (e httpError) Error() string { return fmt.Sprintf("http status %d", e.Code) }

func (e httpError) Is(target error) bool { return e.Code == 404 && target == errNotFound }

// customTypes shows error types that carry details, and how to get them back out of a chain.
func customTypes() {
	fmt.Println("\n--- custom error types, errors.As and errors.AsType ---")

	// Define a type when the caller needs DETAILS: a position, a status code, a retry delay.
	// Do not over-engineer - most errors need no type at all.
	err := fmt.Errorf("compiling template: %w", &parseError{Line: 12, Column: 5, Token: "{{", Err: errNotFound})
	fmt.Println(" ", err)

	// errors.As walks the chain for an error assignable to *target, stores it, reports true.
	// The second argument must be a POINTER to the wanted type; anything else panics on purpose:
	var pe *parseError
	if errors.As(err, &pe) {
		fmt.Printf("  errors.As: line=%d column=%d token=%q\n", pe.Line, pe.Column, pe.Token)
	}
	//	errors.As(err, pe) // go vet: second argument to errors.As must be a non-nil pointer to either a type that implements error, or to any interface type
	//	                   // panic: errors: target must be a non-nil pointer

	// Go 1.26 errors.AsType[T]: generic, no pre-declared variable, no & - prefer it in new code.
	if pe, ok := errors.AsType[*parseError](err); ok {
		fmt.Printf("  errors.AsType[*parseError]: line=%d\n", pe.Line)
	}
	if _, ok := errors.AsType[*retryableError](err); !ok {
		fmt.Println("  errors.AsType[*retryableError]: no match")
	}
	// Is and As compose: parseError's Unwrap lets Is reach the sentinel inside it.
	fmt.Println("  errors.Is(err, errNotFound) through parseError.Unwrap:", errors.Is(err, errNotFound))

	// The idiomatic dispatch: AsType for payloads, Is for sentinels, a default last.
	for _, e := range []error{
		&retryableError{AfterSeconds: 30, Err: errors.New("upstream busy")},
		&parseError{Line: 1, Column: 1, Token: "x", Err: errors.New("unexpected")},
		validationError{Field: "email", Value: "no-at-sign"},
		fmt.Errorf("wrapped: %w", errPermission),
		errors.New("disk on fire"),
	} {
		fmt.Println(" ", handle(e))
	}

	// Pointer or value receiver: with a pointer-receiver Error() only *T is an error, so you
	// must search for *T - mixing them makes As "mysteriously" miss. Pick one and stay with it:
	//	errors.AsType[parseError](err) // ERROR: parseError does not satisfy error (method Error has pointer receiver)
	var ve validationError // value receiver: search for the value type
	wrapped := fmt.Errorf("outer: %w", validationError{Field: "age"})
	fmt.Println("  value-receiver type found as a value:", errors.As(wrapped, &ve))
}

type parseError struct {
	Line, Column int
	Token        string
	Err          error // the wrapped cause
}

func (e *parseError) Error() string {
	return fmt.Sprintf("parse error at %d:%d near %q: %v", e.Line, e.Column, e.Token, e.Err)
}

// Unwrap makes the cause reachable by errors.Is and errors.As.
func (e *parseError) Unwrap() error { return e.Err }

type retryableError struct {
	AfterSeconds int
	Err          error
}

func (e *retryableError) Error() string {
	return fmt.Sprintf("temporary failure, retry after %ds: %v", e.AfterSeconds, e.Err)
}

func (e *retryableError) Unwrap() error { return e.Err }

func handle(err error) string {
	if re, ok := errors.AsType[*retryableError](err); ok {
		return fmt.Sprintf("retryable  -> retry in %ds", re.AfterSeconds)
	}
	if pe, ok := errors.AsType[*parseError](err); ok {
		return fmt.Sprintf("parse      -> report position %d:%d", pe.Line, pe.Column)
	}
	if ve, ok := errors.AsType[validationError](err); ok {
		return fmt.Sprintf("validation -> highlight the %q field", ve.Field)
	}
	if errors.Is(err, errPermission) {
		return "permission -> return 403"
	}
	return fmt.Sprintf("unknown    -> log and return 500 (%v)", err)
}

// inPractice shows the habits that keep error handling readable.
func inPractice() {
	fmt.Println("\n--- error handling in practice ---")

	// Handle an error exactly ONCE - log it or return it, never both, or one failure becomes
	// six log lines. Libraries return; the top of the program logs. Use early returns so the
	// happy path stays unindented (module 004).
	if err := processOrder("ORD-1", "abc"); err != nil {
		fmt.Println(" ", err)
		fmt.Println("    the caller can still branch: Is(strconv.ErrSyntax) =", errors.Is(err, strconv.ErrSyntax))
	}

	// An operation that can partly succeed returns BOTH the partial result and the error - and
	// documents it, like io.Reader.
	parsed, err := parseAll([]string{"1", "2", "x", "4"})
	fmt.Printf("  partial success: parsed=%v err=%v\n", parsed, err)

	// Cleanup that can fail (Close of a buffered file) must reach a named result - a bare
	// `defer f.Close()` loses exactly this error (module 004).
	fmt.Println("  close error via a named result:", writeReport())

	// panic/recover is not error handling: recover only at a boundary (handler, worker loop)
	// and turn it into an error there (module 005).

	// `if err != nil { return nil, err }` is verbose on purpose. The check/handle, try and ?
	// proposals were all rejected - live with it, do not build a framework to hide it.
}

func processOrder(id, qty string) error {
	n, err := strconv.Atoi(qty)
	if err != nil {
		return fmt.Errorf("process order %s: parse quantity: %w", id, err)
	}
	if n <= 0 {
		return fmt.Errorf("process order %s: quantity must be positive, got %d", id, n)
	}
	return nil
}

// parseAll returns everything it managed to parse AND the failure - a documented partial result.
func parseAll(inputs []string) ([]int, error) {
	out := make([]int, 0, len(inputs))
	for i, in := range inputs {
		n, err := strconv.Atoi(in)
		if err != nil {
			return out, fmt.Errorf("element %d: %w", i, err)
		}
		out = append(out, n)
	}
	return out, nil
}

// failingCloser stands in for a buffered file whose final flush hits a full disk.
type failingCloser struct{ io.Writer }

func (failingCloser) Close() error { return errors.New("flush: disk full") }

// writeReport's writes succeed; the failure only appears at Close.
func writeReport() (err error) {
	var f io.WriteCloser = failingCloser{io.Discard}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close report: %w", cerr)
		}
	}()
	if _, err := io.WriteString(f, "report body\n"); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}
