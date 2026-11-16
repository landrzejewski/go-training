// Command 013_stdlib_essentials is module 013 of the Go course: the standard-library packages
// almost every program uses - fmt, strings, time, os/io, encoding/json (v1 and v2), log/slog,
// regexp and uuid.
//
// sync and context are in module 011, slices/maps/cmp/iter in 012, testing in 014.
//
//	go run ./basics/013_stdlib_essentials
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"uuid"
)

func main() {
	formatting()        // fmt verbs, width/precision, Print vs Println, Stringer
	stringsAndBytes()   // Cut/CutLast, Builder, Replacer, bytes.Buffer, strconv
	timeAndDurations()  // the reference layout, Duration, comparing instants
	filesAndIO()        // os.ReadFile/WriteFile, bufio, io composition, os.Root
	jsonV1()            // tags, the float64 trap, unknown and absent fields, streaming
	jsonV2()            // encoding/json/v2 and jsontext (Go 1.27)
	structuredLogging() // log/slog handlers, With, groups, MultiHandler
	regexpAndUUID()     // RE2, named groups, uuid v4 and v7 (Go 1.27)
}

// formatting shows the fmt package: the verbs, width and precision, and the Stringer hook.
func formatting() {
	fmt.Println("--- fmt ---")

	// Three families share one verb set: Print/Println/Printf (stdout), Sprint... (return a
	// string), Fprint... (any io.Writer); plus Errorf (module 009) and the Scan family.
	p := point{X: 1, Y: 2, name: "origin-ish"}
	fmt.Printf("%%v   %v\n", p)  // default format
	fmt.Printf("%%+v  %+v\n", p) // with field names
	fmt.Printf("%%#v  %#v\n", p) // Go syntax
	fmt.Printf("%%T   %T\n", p)  // the TYPE, not the value

	n, f := 255, 1234.5678
	fmt.Printf("%%d=%d %%b=%b %%o=%o %%x=%x %%X=%X %%c=%c\n", n, n, n, n, n, 65)
	fmt.Printf("%%f=%f %%.2f=%.2f %%e=%e %%g=%g\n", f, f, f, f) // %g: shorter of %f/%e
	s := "quoted\ttext"
	fmt.Printf("%%s=%s %%q=%q %%t=%t %%p=%p\n", s, s, true, &n)

	// Width.precision: %8.2f; %-10s left-aligns; %08d zero-pads; * takes the width from an arg.
	fmt.Printf("|%8.2f| |%-10s| |%08d| |%*d|\n", f, "left", 42, 6, 7)

	// Print adds a space only between two operands when NEITHER is a string; Println always
	// adds spaces and a newline. This asymmetry surprises people constantly.
	fmt.Print("Print:   ", 1, 2, "a", "b", 3, "\n")
	fmt.Println("Println:", 1, 2, "a", "b", 3)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Fprintf writes to any io.Writer")
	fmt.Printf("Sprintf: %q  Fprintf: %q\n", fmt.Sprintf("%s-%03d", "id", 7), buf.String())

	// %v on a fmt.Stringer calls String(); on an error it calls Error().
	fmt.Printf("a Stringer is used by %%v: %v\n", celsius(21.5))

	var a int
	var word string
	_, _ = fmt.Sscanf("42 gopher", "%d %s", &a, &word)
	fmt.Printf("Sscanf parsed: %d %q\n", a, word)

	// go vet's printf check matches every verb against its argument (and understands your own
	// ...f wrappers) - one of the highest-value checks in the toolchain. For hot paths strconv
	// beats fmt substantially (module 002a).
}

type point struct {
	X, Y int
	name string
}

type celsius float64

// String makes celsius a fmt.Stringer. Never format the receiver itself with %v in here -
// Sprintf would call String again, forever. Convert to the underlying type first:
//
//	return fmt.Sprintf("%v", c) // vet: fmt.Sprintf format %v with arg c causes recursive (training.pl/go/basics/013_stdlib_essentials.celsius).String method call
func (c celsius) String() string { return fmt.Sprintf("%.1f°C", float64(c)) }

// stringsAndBytes shows the strings, bytes and strconv packages.
func stringsAndBytes() {
	fmt.Println("\n--- strings, strconv and bytes ---")

	s := "Go is expressive, concise, clean, and efficient"
	fmt.Printf("Contains=%t HasPrefix=%t Index=%d Count(\", \")=%d\n",
		strings.Contains(s, "clean"), strings.HasPrefix(s, "Go"),
		strings.Index(s, "concise"), strings.Count(s, ", "))
	fmt.Printf("Fields: %q\n", strings.Fields("  split   on   whitespace  "))
	fmt.Printf("Split/Join: %q\n", strings.Join(strings.Split("a,b,c", ","), " | "))
	fmt.Printf("EqualFold(\"Go\", \"GO\")=%t (case-insensitive, Unicode-aware)\n",
		strings.EqualFold("Go", "GO"))
	fmt.Printf("TrimSpace=%q TrimPrefix=%q\n",
		strings.TrimSpace("  padded  "), strings.TrimPrefix("prefix-value", "prefix-"))
	// strings.Title is deprecated - use golang.org/x/text/cases.

	// Cut (Go 1.18) is usually what you want instead of Split: it splits at the FIRST separator
	// and reports whether it was there.
	before, after, found := strings.Cut("key=value=extra", "=")
	fmt.Printf("Cut at the FIRST separator: before=%q after=%q found=%t\n", before, after, found)

	// CutLast (Go 1.27) is its mirror - before 1.27 this took LastIndex plus two slicings.
	dir, base, found := strings.CutLast("/usr/local/bin/go", "/")
	fmt.Printf("CutLast at the LAST separator: dir=%q base=%q found=%t\n", dir, base, found)
	name, ext, _ := strings.CutLast("archive.tar.gz", ".")
	fmt.Printf("CutLast for an extension: name=%q ext=%q\n", name, ext)

	// CutPrefix / CutSuffix (Go 1.20) replace the HasPrefix + slice pair.
	if rest, ok := strings.CutPrefix("v1.27.0", "v"); ok {
		fmt.Printf("CutPrefix: %q\n", rest)
	}

	// strings.Builder for building in a loop (+= is quadratic). It is an io.Writer, and must
	// not be copied after first use.
	var sb strings.Builder
	sb.Grow(64) // preallocate when you roughly know the size
	for i := range 4 {
		fmt.Fprintf(&sb, "[%d]", i)
	}
	fmt.Printf("strings.Builder: %q\n", sb.String())

	// NewReplacer does many replacements in a single pass and is safe for concurrent use.
	// (strings.NewReader turns a string into an io.Reader - see filesAndIO.)
	r := strings.NewReplacer("<", "&lt;", ">", "&gt;", "&", "&amp;")
	fmt.Printf("NewReplacer: %q\n", r.Replace("<b>a & b</b>"))

	// bytes mirrors strings for []byte - use it when you already have bytes, since converting
	// to string copies. bytes.Buffer is the read/write counterpart of strings.Builder.
	b := []byte("bytes mirrors strings")
	fmt.Printf("bytes.Contains=%t bytes.ToUpper=%q\n",
		bytes.Contains(b, []byte("mirror")), bytes.ToUpper(b[:5]))
	buf := bytes.NewBufferString("peek at me")
	peeked, _ := buf.Peek(4) // Go 1.26: look without consuming
	fmt.Printf("Buffer.Peek (Go 1.26): %q, buffer still has %d bytes\n", peeked, buf.Len())

	// strconv converts numbers and strings (module 002a). Never string(someInt) - that is a rune.
	fmt.Printf("strconv.Quote=%s Itoa(42)=%q FormatBool(true)=%q\n",
		strconv.Quote("tab\there"), strconv.Itoa(42), strconv.FormatBool(true))
}

// timeAndDurations shows time.Time (an instant) and time.Duration (int64 nanoseconds).
func timeAndDurations() {
	fmt.Println("\n--- time ---")

	t := time.Date(2026, time.August, 30, 14, 5, 6, 0, time.UTC) // fixed, so output is stable
	fmt.Println("the instant:", t)

	// The layout is not %Y-%m-%d but the reference DATE Mon Jan 2 15:04:05 MST 2006, i.e.
	// 01/02 03:04:05PM '06 -0700 - the digits 1 to 7. Prefer constants such as time.RFC3339.
	fmt.Println("RFC3339: ", t.Format(time.RFC3339))
	fmt.Println("custom:  ", t.Format("2006-01-02 15:04:05"))
	fmt.Println("friendly:", t.Format("Mon, 02 Jan 2006 at 3:04 PM"))
	parsed, err := time.Parse(time.RFC3339, "2026-08-30T14:05:06Z")
	fmt.Printf("Parse round trip: %v (err=%v, Equal=%t)\n", parsed, err, parsed.Equal(t))

	d := 90 * time.Minute
	fmt.Printf("90*time.Minute = %v = %.1f hours = %d ns\n", d, d.Hours(), d.Nanoseconds())
	fmt.Println(`ParseDuration("1h30m") =`, mustDuration("1h30m"))
	fmt.Println("Truncate/Round to the hour:", (93 * time.Minute).Truncate(time.Hour), (93 * time.Minute).Round(time.Hour))

	// Duration is an integer type: an untyped constant adapts (5 * time.Second), an int
	// VARIABLE needs a conversion.
	n := 5
	fmt.Println("time.Duration(n) * time.Second =", time.Duration(n)*time.Second)
	//	fmt.Println(n * time.Second) // ERROR: invalid operation: n * time.Second (mismatched types int and time.Duration)

	// Never compare instants with ==: a Time holds a wall clock, a monotonic reading and a
	// location, so the same moment can compare unequal. Use Equal, Before, After, Compare.
	later := t.Add(36 * time.Hour)
	fmt.Printf("Add(36h) = %s, Sub = %v\n", later.Format(time.RFC3339), later.Sub(t))
	fmt.Printf("Before=%t After=%t Compare=%d\n", t.Before(later), t.After(later), t.Compare(later))

	fmt.Printf("Year=%d Month=%v Day=%d Weekday=%v YearDay=%d\n",
		t.Year(), t.Month(), t.Day(), t.Weekday(), t.YearDay())
	if warsaw, err := time.LoadLocation("Europe/Warsaw"); err == nil {
		fmt.Println("in Europe/Warsaw:", t.In(warsaw).Format(time.RFC3339))
	}

	// time.Now() carries a MONOTONIC reading, so time.Since is immune to wall-clock changes.
	// Add keeps it; Round, Truncate, UTC, Local and In strip it.
	start := time.Now()
	time.Sleep(2 * time.Millisecond)
	fmt.Println("time.Since (monotonic):", time.Since(start).Round(time.Millisecond))

	// Timers: before Go 1.23 an unstopped time.After/time.Tick timer was never collected (a
	// leak in hot loops); now it is, but time.NewTimer/NewTicker + Stop still frees it promptly.
	// To test timeouts deterministically, use testing/synctest's fake clock (module 014).
}

func mustDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}

// filesAndIO shows os, io, bufio and the path-traversal-safe os.Root.
func filesAndIO() {
	fmt.Println("\n--- os, io, bufio and os.Root ---")

	// LookupEnv distinguishes "unset" from "set to empty"; Getenv returns "" for both.
	_, homeSet := os.LookupEnv("HOME")
	_, otherSet := os.LookupEnv("DEFINITELY_NOT_SET_12345")
	fmt.Printf("os.Args[0]=%s  LookupEnv: HOME set=%t, unknown set=%t\n", filepath.Base(os.Args[0]), homeSet, otherSet)

	dir, err := os.MkdirTemp("", "stdlib-*")
	if err != nil {
		fmt.Println("could not create a temp dir:", err)
		return
	}
	// os.Exit does NOT run deferred functions like this one - call it only at the very top of
	// main, never in a library.
	defer os.RemoveAll(dir)

	// os.WriteFile / os.ReadFile: your default for anything that fits in memory.
	path := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(path, []byte("first line\nsecond line\nthird line\n"), 0o644); err != nil {
		fmt.Println("write failed:", err)
		return
	}
	data, _ := os.ReadFile(path)
	info, _ := os.Stat(path)
	fmt.Printf("WriteFile + ReadFile: %d bytes; Stat: name=%s mode=%v\n", len(data), info.Name(), info.Mode())

	// Stream with os.Open + bufio.Scanner. Its default line limit is 64 KB (raise it with
	// scanner.Buffer). A bufio.Writer must be Flushed - and that error checked.
	f, _ := os.Open(path)
	scanner := bufio.NewScanner(f)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	_ = f.Close()
	fmt.Printf("bufio.Scanner read %d lines (err=%v)\n", lines, scanner.Err())

	// io is the interface layer (Reader, Writer, Copy, ReadAll, io.EOF, ...) and its helpers
	// compose readers and writers.
	var captured bytes.Buffer
	consumed, _ := io.ReadAll(io.TeeReader(strings.NewReader("teed"), &captured))
	fmt.Printf("io.TeeReader: consumed=%q, copy captured=%q\n", consumed, captured.String())
	limited, _ := io.ReadAll(io.LimitReader(strings.NewReader("truncate me here"), 8))
	fmt.Printf("io.LimitReader(8): %q\n", limited)
	var fanout bytes.Buffer
	fmt.Fprint(io.MultiWriter(&fanout, io.Discard), "one write, several writers")
	fmt.Printf("io.MultiWriter: %q\n", fanout.String())

	// os.Root (Go 1.24; ReadFile, WriteFile, MkdirAll, ... added in 1.25) confines every
	// operation to one directory. "..", absolute paths and escaping symlinks are refused by
	// the OS, not by string checks - use it whenever a path comes from outside your program.
	root, err := os.OpenRoot(dir)
	if err != nil {
		fmt.Println("OpenRoot failed:", err)
		return
	}
	defer root.Close()
	inside, err := root.ReadFile("notes.txt")
	fmt.Printf("root.ReadFile(\"notes.txt\"): %d bytes, err=%v\n", len(inside), err)
	_, err = root.Open("../../../etc/passwd")
	fmt.Println(`root.Open("../../../etc/passwd") ->`, err)

	// io/fs is the read-only filesystem abstraction (fs.FS, fs.WalkDir, fs.Sub). embed.FS,
	// os.DirFS and fstest.MapFS all satisfy it - that is what makes //go:embed testable.
}

// jsonV1 shows encoding/json, its struct tags and its traps.
func jsonV1() {
	fmt.Println("\n--- encoding/json (v1) ---")

	// Marshal/Unmarshal work on whole values; NewEncoder/NewDecoder on streams. Only EXPORTED
	// fields are visible - no tag changes that. Tag options: rename, "-", omitempty, ",string",
	// and omitzero (Go 1.24), which omits a zero struct or time.Time where omitempty cannot.
	u := user{ID: 1, Name: "Ada", Internal: "hidden", password: "secret"}
	b, _ := json.Marshal(u)
	fmt.Printf("Marshal: %s  <- no email/createdAt (omitempty/omitzero), no Internal (-), no password (unexported)\n", b)
	indented, _ := json.MarshalIndent(user{ID: 2, Name: "Grace",
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, "", "  ")
	fmt.Printf("MarshalIndent:\n%s\n", indented)

	// Unmarshalling into any yields only bool, float64 (for EVERY number), string, nil, []any
	// and map[string]any - so obj["id"].(int) panics.
	var loose any
	_ = json.Unmarshal([]byte(`{"id":1,"ratio":0.5,"tags":["a"],"nested":{"k":true}}`), &loose)
	obj := loose.(map[string]any)
	fmt.Printf("into any: id is %T, tags %T, nested %T\n", obj["id"], obj["tags"], obj["nested"])

	// Decoder.UseNumber keeps the exact text as a json.Number.
	dec := json.NewDecoder(strings.NewReader(`{"big": 123456789012345678901}`))
	dec.UseNumber()
	var exact map[string]any
	_ = dec.Decode(&exact)
	fmt.Printf("UseNumber keeps precision: %v (%T)\n", exact["big"], exact["big"])

	// Unknown fields are silently ignored by default - always disallow them for config files.
	var target user
	strict := json.NewDecoder(strings.NewReader(`{"id":1,"typo":"oops"}`))
	strict.DisallowUnknownFields()
	fmt.Println("DisallowUnknownFields:", strict.Decode(&target))

	// Missing fields stay at their zero value; a pointer field tells "absent" from "zero"
	// (json.RawMessage is the other option).
	type patch struct {
		Name *string `json:"name"`
	}
	var absent, explicit patch
	_ = json.Unmarshal([]byte(`{}`), &absent)
	_ = json.Unmarshal([]byte(`{"name":""}`), &explicit)
	fmt.Printf("absent name is %v, explicitly empty is %q\n", absent.Name, *explicit.Name)

	// An Encoder writes one JSON value per line (newline-delimited JSON).
	var streamed bytes.Buffer
	enc := json.NewEncoder(&streamed)
	for _, v := range []user{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}} {
		_ = enc.Encode(v)
	}
	fmt.Print("Encoder output:\n", indent(streamed.String(), "  "))

	// Custom formats: implement json.Marshaler/Unmarshaler, or better encoding.TextMarshaler,
	// which also serves XML and map keys (module 008). v1's weaknesses - slow, no easy
	// streaming of big arrays, poor errors, case-insensitive matching, odd omitempty - are what
	// v2 fixes.
}

type user struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitzero"` // Go 1.24: omitted when zero
	Internal  string    `json:"-"`
	password  string
}

func indent(s, prefix string) string {
	var sb strings.Builder
	for line := range strings.SplitSeq(strings.TrimRight(s, "\n"), "\n") {
		sb.WriteString(prefix + line + "\n")
	}
	return sb.String()
}

// jsonV2 shows encoding/json/v2 and encoding/json/jsontext, new in Go 1.27.
func jsonV2() {
	fmt.Println("\n--- encoding/json/v2 and jsontext (Go 1.27) ---")

	// A NEW package, not a breaking change: encoding/json keeps its behaviour (and is now built
	// on v2). jsontext is the syntax layer - tokens and values, no Go types; json/v2 is the
	// semantic layer on top. v1 is not deprecated; use v2 for speed, strictness or streaming.

	// Options are arguments, not decoder state, and they compose (WithIndent replaces
	// MarshalIndent).
	cfg := config{Name: "api", Port: 8080, Timeout: timeout(30 * time.Second)}
	plain, _ := jsonv2.Marshal(cfg)
	fmt.Printf("v2 Marshal: %s\n", plain)
	indented, _ := jsonv2.Marshal(cfg, jsontext.WithIndent("  "))
	fmt.Print("v2 WithIndent:\n", indent(string(indented), "  "))
	spaced, _ := jsonv2.Marshal(cfg, jsontext.SpaceAfterColon(true), jsontext.SpaceAfterComma(true))
	fmt.Printf("options compose: %s\n", spaced)

	// v2 refuses to guess: v1 silently wrote a time.Duration as nanoseconds (undocumented, a
	// meaningless integer to read back); v2 makes you choose - here via the timeout wrapper.
	// (The format: tag option that will cover this is not enabled yet in 1.27.)
	type bare struct {
		Timeout time.Duration `json:"timeout"`
	}
	_, durErr := jsonv2.Marshal(bare{Timeout: 30 * time.Second})
	fmt.Println("v2 on a raw time.Duration:", durErr)
	v1Bytes, _ := json.Marshal(bare{Timeout: 30 * time.Second})
	fmt.Printf("v1 silently emitted nanoseconds: %s\n", v1Bytes)
	var roundTrip config
	_ = jsonv2.Unmarshal([]byte(`{"name":"api","port":1,"timeout":"1m30s"}`), &roundTrip)
	fmt.Println("the wrapper round-trips as readable text:", time.Duration(roundTrip.Timeout))

	// Field matching is case-SENSITIVE by default (v1's case-insensitivity surprised people).
	var strictTarget, looseTarget, v1Target config
	strictErr := jsonv2.Unmarshal([]byte(`{"NAME":"x","port":1}`), &strictTarget)
	fmt.Printf("v2 is case-sensitive: name=%q (err=%v)\n", strictTarget.Name, strictErr)
	_ = jsonv2.Unmarshal([]byte(`{"NAME":"x","port":1}`), &looseTarget, jsonv2.MatchCaseInsensitiveNames(true))
	fmt.Printf("with MatchCaseInsensitiveNames(true): name=%q\n", looseTarget.Name)
	_ = json.Unmarshal([]byte(`{"NAME":"x"}`), &v1Target)
	fmt.Printf("v1 was case-insensitive: name=%q\n", v1Target.Name)

	// Duplicate names (and invalid UTF-8) are errors in v2, not silent defaults.
	dupErr := jsonv2.Unmarshal([]byte(`{"port":1,"port":2}`), &config{})
	v1DupErr := json.Unmarshal([]byte(`{"port":1,"port":2}`), &config{})
	fmt.Printf("duplicate names: v2 rejects=%t, v1 rejects=%t (last one wins)\n", dupErr != nil, v1DupErr != nil)

	// jsontext: reformat, canonicalise or validate JSON without unmarshalling it.
	value := jsontext.Value(`{"b":2,"a":{"d":4,"c":3}}`)
	_ = value.Canonicalize() // sorts object names, compacts
	fmt.Printf("jsontext.Value.Canonicalize: %s\n", value)
	pretty := jsontext.Value(`{"a":[1,2],"b":null}`)
	_ = pretty.Indent(jsontext.WithIndent("  "))
	fmt.Print("jsontext.Value.Indent:\n", indent(string(pretty), "  "))

	// Token-level streaming with no Go type at all - how you walk a huge array without holding
	// it in memory. Custom types can stream too: MarshalJSONTo(*jsontext.Encoder) and
	// UnmarshalJSONFrom(*jsontext.Decoder) instead of allocating a []byte.
	dec := jsontext.NewDecoder(strings.NewReader(`{"id":1,"tags":["a","b"]}`))
	var kinds []string
	for {
		tok, err := dec.ReadToken()
		if err != nil {
			break
		}
		kinds = append(kinds, tok.Kind().String())
	}
	fmt.Println("jsontext token stream:", strings.Join(kinds, " "))

	// Migrating? v1 gained DefaultOptionsV1() and ...WithLegacySemantics options, so v1
	// behaviour can be opted into piece by piece.
}

type config struct {
	Name     string  `json:"name"`
	Port     int     `json:"port"`
	Timeout  timeout `json:"timeout"`
	Replicas int     `json:"replicas,omitzero"`
}

// timeout gives time.Duration a TEXT representation - how a duration travels through json/v2.
type timeout time.Duration

func (d timeout) MarshalText() ([]byte, error) { return []byte(time.Duration(d).String()), nil }

func (d *timeout) UnmarshalText(b []byte) error {
	parsed, err := time.ParseDuration(string(b))
	if err != nil {
		return fmt.Errorf("timeout: %w", err)
	}
	*d = timeout(parsed)
	return nil
}

// structuredLogging shows log/slog (Go 1.21), the standard structured logger.
func structuredLogging() {
	fmt.Println("\n--- log/slog ---")

	// The old log package is fine for a CLI; anything shipping logs to a system uses slog.
	// HandlerOptions sets the minimum Level, AddSource and a ReplaceAttr hook (redact or rename
	// keys) - here it drops the time so the output is reproducible.
	stripTime := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}
	text := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, ReplaceAttr: stripTime,
	}))

	// Levels are numbers (Debug=-4, Info=0, Warn=4, Error=8), so custom levels fit between.
	// Attributes: loose key/value pairs (vet checks odd counts) or the typed slog.String,
	// slog.Int, slog.Duration, slog.Any - faster and cannot get out of step. Prefer typed.
	text.Info("server started", "port", 8080, "tls", false)
	text.Warn("high latency", slog.Duration("p99", 250*time.Millisecond))
	text.Error("request failed", slog.String("path", "/api"), slog.Int("status", 500))

	// With returns a logger that adds attributes to every record (the way to attach a request
	// ID); WithGroup nests later attributes under a name.
	reqLogger := text.With(slog.String("requestID", "req-42"), slog.String("user", "ada"))
	reqLogger.Info("handling request")
	reqLogger.Info("request complete", slog.Int("ms", 12))
	text.WithGroup("db").Info("query", slog.String("table", "users"), slog.Int("rows", 3))

	// slog.GroupAttrs (Go 1.25) builds a group from a []slog.Attr. Prefer the Context variants
	// (InfoContext, LogAttrs with ctx) so a handler can pull trace IDs from the context - and
	// never pass a nil Context (module 011).
	attrs := []slog.Attr{slog.String("region", "eu"), slog.Int("shard", 2)}
	text.LogAttrs(context.Background(), slog.LevelInfo, "placement", slog.GroupAttrs("location", attrs...))

	// The JSON handler is for log aggregation.
	jsonLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: stripTime}))
	jsonLogger.Info("structured", slog.String("env", "prod"), slog.Int("version", 27))

	// slog.NewMultiHandler (Go 1.26) fans one record out to several handlers - text to the
	// console and JSON to a file, with no third-party package.
	var captured bytes.Buffer
	multi := slog.New(slog.NewMultiHandler(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: stripTime}),
		slog.NewJSONHandler(&captured, &slog.HandlerOptions{ReplaceAttr: stripTime}),
	))
	multi.Info("fanned out", slog.String("to", "two handlers"))
	fmt.Print("MultiHandler's JSON buffer received: ", captured.String())
}

// Compile a regexp once, at package level. MustCompile panics, which is right for a pattern
// that is effectively a constant; compiling inside a loop is a common, expensive mistake.
var (
	emailRe = regexp.MustCompile(`^[\w.+-]+@[\w-]+\.[\w.]+$`)
	logRe   = regexp.MustCompile(`^(?P<level>\w+)\s+(?P<time>[\d:]+)\s+(?P<msg>.*)$`)
)

// regexpAndUUID shows the RE2-based regexp package and the uuid package (Go 1.27).
func regexpAndUUID() {
	fmt.Println("\n--- regexp and uuid ---")

	for _, e := range []string{"ada@example.com", "not-an-email", "a.b+c@sub.example.co.uk"} {
		fmt.Printf("MatchString(%-25q) = %t\n", e, emailRe.MatchString(e))
	}

	// Names are systematic: Find[All][String][Submatch][Index]. Submatch = capture groups,
	// Index = byte offsets, All takes a limit n (-1 = all). Named groups (?P<name>...) plus
	// SubexpNames make a match self-documenting. A *Regexp is safe for concurrent use.
	if m := logRe.FindStringSubmatch("ERROR 14:05:06 database connection lost"); m != nil {
		for i, name := range logRe.SubexpNames() {
			if i > 0 {
				fmt.Printf("  group %-5s = %q\n", name, m[i])
			}
		}
	}
	words := regexp.MustCompile(`\w+`)
	fmt.Println("FindAllString(-1):", words.FindAllString("one two three", -1))
	fmt.Println("FindAllString(2): ", words.FindAllString("one two three", 2))
	fmt.Printf("ReplaceAllString: %q  ReplaceAllStringFunc: %q\n",
		regexp.MustCompile(`\d+`).ReplaceAllString("a1b22c333", "#"),
		words.ReplaceAllStringFunc("go is fun", strings.ToUpper))

	// regexp is RE2: linear time, so NO backtracking - no backreferences, lookahead or
	// lookbehind. In exchange catastrophic backtracking (a ReDoS class of bug) cannot happen.
	// For a fixed substring, strings.Contains is far faster than any regexp.
	_, err := regexp.Compile(`(a)\1`)
	fmt.Println("backreference:", err)

	// uuid (Go 1.27, RFC 9562) replaces github.com/google/uuid. Random bits come from a
	// cryptographically secure source. New() is the general-purpose choice; NewV4 is 122
	// random bits.
	v4 := uuid.NewV4()
	fmt.Println("uuid.NewV4():", v4)

	// NewV7 starts with a 48-bit millisecond timestamp, so values sort chronologically - use it
	// for database keys; random v4 keys scatter B-tree inserts and ruin index locality.
	var seq []uuid.UUID
	for range 3 {
		seq = append(seq, uuid.NewV7())
		time.Sleep(2 * time.Millisecond)
	}
	for i, id := range seq {
		fmt.Printf("uuid.NewV7(): %v  greater than previous=%t\n", id, i == 0 || id.Compare(seq[i-1]) > 0)
	}

	parsed, err := uuid.Parse(v4.String())
	fmt.Printf("Parse round trip: equal=%t err=%v; Nil()=%v\n", parsed == v4, err, uuid.Nil())

	// UUID implements MarshalText/UnmarshalText, so JSON and map keys need no extra code.
	encoded, _ := json.Marshal(map[string]uuid.UUID{"id": parsed})
	fmt.Printf("JSON via MarshalText: %s\n", encoded)
}
