// Command 015_modules_and_tooling is module 015 of the Go course: modules, build constraints,
// //go:embed and the go toolchain.
//
// One toolchain does everything - build, test, format, vet, document, profile, manage
// dependencies and fetch its own newer versions - and apart from go.mod there is no config file.
//
//	go run ./basics/015_modules_and_tooling
//	go run -ldflags="-X main.version=1.2.3" ./basics/015_modules_and_tooling
package main

import (
	"embed"
	"fmt"
	"go/build"
	"io/fs"
	"runtime"
	"runtime/debug"
	"strings"
)

func main() {
	modules()              // go.mod, the go line, go.sum, MVS, /v2 import paths
	dependencies()         // go get vs go install, tool deps, workspaces, GOTOOLCHAIN
	buildAndEmbed()        // //go:build, filename suffixes, //go:embed
	goCommand()            // caches, cross-compiling, -ldflags, PGO, GODEBUG
	formattingAndLinting() // gofmt, go vet, staticcheck
	profiling()            // pprof, tracing, GOMAXPROCS, GOGC, GOMEMLIMIT
	projectLayout()        // no official layout; naming and other conventions
}

// modules shows what a module is, using the build info baked into this very binary.
func modules() {
	fmt.Println("--- modules and go.mod ---")

	// A module is a tree of packages with go.mod at its root. Its path is the prefix of every
	// import path inside it: this repo is training.pl/go, so this program's package path is
	// training.pl/go/basics/015_modules_and_tooling. The go.mod directives:
	//	module training.pl/go        the module path
	//	go 1.27                      the LANGUAGE version, not the toolchain version
	//	toolchain go1.27.0           the minimum toolchain, added automatically when needed
	//	require ( ... )              dependencies; transitive ones marked // indirect
	//	replace old => new           redirect a path, e.g. to a local checkout
	//	exclude mod v1.2.3           never select this version
	//	retract v1.0.1               (your own module) tell users a release is bad
	//	tool golang.org/x/tools/...  Go 1.24: a tool dependency
	if info, ok := debug.ReadBuildInfo(); ok {
		fmt.Println("package path:", info.Path)
		fmt.Println("main module: ", info.Main.Path, info.Main.Version)
		fmt.Println("built with:  ", info.GoVersion)
		// Only modules this binary actually imports are linked in - this one uses just the
		// standard library, although go.mod requires gin, gorm and pq for other modules.
		fmt.Println("dependency modules linked into this binary:", len(info.Deps))
		for _, s := range info.Settings {
			switch s.Key {
			case "GOOS", "GOARCH", "CGO_ENABLED", "-ldflags", "vcs.revision", "vcs.modified":
				fmt.Printf("  build setting %-12s = %s\n", s.Key, s.Value) // go build also records VCS state
			}
		}
	}

	// The go line selects language semantics PER MODULE: a module saying go 1.21 keeps the
	// pre-1.22 loop-variable semantics even when compiled by Go 1.27. That is how Go changes the
	// language without breaking its compatibility promise.
	//
	// go.sum records a hash of every module version used. It is a TAMPER CHECK, not a lock file -
	// go.mod already pins versions. Commit both.
	//
	// Minimal Version Selection picks the HIGHEST version anyone in the graph explicitly asks for,
	// and no higher: no solver, reproducible without a lock file, and adding a dependency cannot
	// silently upgrade an unrelated one.
	//
	// Semantic import versioning: v2+ puts the major version IN THE PATH (github.com/foo/bar/v2),
	// so v1 and v2 are different packages and can coexist - hence math/rand/v2, encoding/json/v2.
	// internal/ is enforced: only code rooted at the parent of internal/ may import it.
	//
	//	go mod tidy      add what is used, drop what is not - run before every commit
	//	go mod why -m X  why is this dependency here?
	//	go list -m all   the full selected version graph
	//	go mod graph / go mod verify / go mod vendor
}

// dependencies shows the toolchain version and how dependencies, tools and workspaces work.
func dependencies() {
	fmt.Println("\n--- dependencies, tools and workspaces ---")

	fmt.Printf("this toolchain: %s on %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Println("GOPATH (module cache lives under it):", build.Default.GOPATH)

	//	go get example.com/pkg@latest   add or upgrade (@v1.2.3 a version, @none removes it)
	//	go get -u ./...                 upgrade everything - carefully
	//	go install example.com/cmd@v1   build and install a BINARY without touching go.mod
	// Since Go 1.18 go get only edits go.mod; it no longer builds or installs anything.

	// Tool dependencies (Go 1.24) replace the old tools.go hack (a //go:build tools file that
	// imported packages only so go mod tidy kept them):
	//	go get -tool golang.org/x/tools/cmd/stringer   adds a tool line to go.mod
	//	go tool stringer -type=weekday                 runs exactly the pinned version
	//	go tool                                        lists the available tools
	// stringer would generate the String method that weekday lacks:
	fmt.Printf("weekday without a String method prints as a number: %v\n", wednesday)

	// Workspaces (Go 1.18) develop several modules together:
	//	go work init ./api ./worker ./shared
	//	go work use ./newmodule
	// go.work is LOCAL state and normally not committed - unlike a replace in go.mod, which
	// ships to your users.
	//
	// A nested go.mod CUTS its subtree out of the parent module: go build ./... at the root
	// stops descending into it and go list ./... never lists it - which is how a nested module
	// goes stale unnoticed. Use one module (or a workspace) unless the subtree is published alone.
	//
	// go mod vendor copies dependencies into vendor/, which builds then use automatically -
	// worth it for hermetic or air-gapped builds, a nuisance otherwise.
	//
	// Since Go 1.21 the toolchain line can require a newer Go than the one installed, and
	// GOTOOLCHAIN=auto (the default) downloads and runs it; GOTOOLCHAIN=local refuses.
}

type weekday int

const (
	sunday weekday = iota
	monday
	tuesday
	wednesday
)

// The //go:embed directive compiles files INTO the binary at build time (Go 1.16). The variable
// must be at package level and of type string, []byte or embed.FS. Paths are relative to this
// file's directory, may not contain .. or escape the module, and skip names starting with . or _
// unless prefixed with all:. Mistakes fail the BUILD, not the deployment:
//
//	//go:embed missing.txt // ERROR: pattern missing.txt: no matching files found
//	//go:embed ../go.mod   // ERROR: pattern ../go.mod: invalid pattern syntax
//	//go:embed embedded.txt
//	var n int // ERROR: go:embed cannot apply to var of type int
//	(the same directive on a local variable) // ERROR: go:embed cannot apply to var inside func
//
// The embed package must be imported, as import _ "embed" when only string/[]byte is used:
//
//	(no import of embed) // ERROR: go:embed requires import "embed" (or import _ "embed", if package is not used)

//go:embed embedded.txt
var embeddedText string

//go:embed *.txt
var assets embed.FS

// buildAndEmbed shows build constraints and //go:embed.
func buildAndEmbed() {
	fmt.Println("\n--- build constraints and go:embed ---")

	// A //go:build line BEFORE the package clause, followed by a blank line, decides whether the
	// file is compiled at all. It is a boolean expression over build tags: GOOS and GOARCH values,
	// go1.N, cgo, race, unix and any custom tag passed with -tags:
	//	//go:build linux && amd64
	//	//go:build (linux || darwin) && !cgo
	//	//go:build integration          <- go test -tags integration ./...
	// No space after //: "// go:build" is an ordinary, ignored comment. The old // +build form is
	// deprecated. Uses: platform-specific code, gating integration tests, experimental code.
	fmt.Printf("build tags satisfied here include: %s\n", strings.Join(activeTags(), ", "))

	// A filename suffix is an implicit constraint and usually cleaner for OS splits:
	// x_linux.go, x_windows_amd64.go, x_test.go. But _unix is NOT a recognised suffix, which is
	// why examples/chat/client/platform_unix.go carries //go:build !windows next to
	// platform_windows.go.

	fmt.Printf("//go:embed into a string: %q\n", strings.TrimSpace(embeddedText))
	// embed.FS satisfies fs.FS, so it works with http.FileServerFS, template.ParseFS, fs.WalkDir.
	names, _ := fs.Glob(assets, "*")
	fmt.Println("//go:embed into an embed.FS:", names)
}

func activeTags() []string {
	tags := []string{runtime.GOOS, runtime.GOARCH}
	switch runtime.GOOS {
	case "linux", "darwin", "freebsd", "openbsd", "netbsd", "dragonfly", "solaris", "aix":
		tags = append(tags, "unix")
	}
	minor, _, _ := strings.Cut(strings.TrimPrefix(runtime.Version(), "go1."), ".")
	return append(tags, "go1."+minor+" and every earlier go1.N")
}

// version is stamped at link time: go build -ldflags="-X main.version=1.2.3".
var version = "dev"

// goCommand shows a value injected by the linker and lists what the go command can do.
func goCommand() {
	fmt.Println("\n--- the go command ---")

	//	go run .                    build and run, leaving no binary
	//	go build ./...              build every package; ./... means "this tree"
	//	go install ./cmd/app        build and place the binary in $GOBIN
	//	go test ./... / go vet ./... / go fmt ./...
	//	go doc net/http Handler     documentation in the terminal
	//	go doc -http=:6060          Go 1.25: the docs in a browser, offline
	//	go version -m ./binary      the module graph baked into a built binary
	//	go generate ./...           run //go:generate directives - NOT part of go build:
	//	                            run it and commit the output (pin the generator with tool)
	//	go clean -cache -testcache  when you suspect a stale cache
	//	go env / go env -w          inspect and persist settings
	// Builds AND test results are cached: an unchanged passing test prints (cached); -count=1
	// forces a real run.

	// -ldflags="-X main.version=1.2.3" sets a string variable at link time - the standard way
	// to stamp a version. -ldflags="-s -w" strips the symbol table and DWARF (~25% smaller).
	fmt.Println("version =", version)

	// Cross-compiling needs no extra toolchain - the standard library is compiled on demand.
	// CGO_ENABLED=0 gives a fully static binary, which is why Go images can be FROM scratch.
	fmt.Printf("cross-compile from %s/%s: GOOS=linux GOARCH=arm64 go build ./...\n", runtime.GOOS, runtime.GOARCH)

	// Profile-guided optimisation (Go 1.21): a default.pgo next to main is used automatically;
	// typically 2-7% faster for no code change.
	//
	// When a release changes a behaviour it adds a GODEBUG setting restoring the old one
	// (GODEBUG=httplaxcontentlength=1, asynctimerchan=1, ...); the go line in go.mod sets the
	// defaults. That is what makes the Go 1 compatibility promise workable in practice.
}

// formattingAndLinting covers gofmt, go vet and third-party linters.
func formattingAndLinting() {
	fmt.Println("\n--- formatting, vetting and linting ---")

	// There is ONE Go style and gofmt is not configurable - no indent width, brace or line-length
	// options - so there are no style debates. Tabs indent, spaces align.
	//	gofmt -l .  list unformatted files (CI)   gofmt -w .  rewrite   gofmt -d .  diff
	//	gofmt -r 'a[b:len(a)] -> a[b:]'           apply a rewrite rule across a codebase
	// goimports also adds and removes imports; it is what most editors run on save.
	fmt.Println("CI: gofmt -l . && go vet ./... && staticcheck ./...")

	// go vet reports code that compiles but is almost certainly wrong, and a subset runs
	// automatically during go test. Checks that earn their keep:
	for _, c := range []struct{ name, what string }{
		{"printf", "verb/argument mismatches, including your own ...f wrappers"},
		{"copylocks", "a sync.Mutex copied by value"},
		{"lostcancel", "a context.CancelFunc not called on every path"},
		{"structtag", "a malformed struct tag"},
		{"unusedresult", "a discarded result of fmt.Sprintf, errors.New, strings.ToUpper"},
		{"loopclosure", "capturing a loop variable (mostly moot since Go 1.22)"},
	} {
		fmt.Printf("  vet %-13s %s\n", c.name, c.what)
	}
	// shadow and fieldalignment are NOT built in: they live in golang.org/x/tools and run via
	// go vet -vettool=$(which shadow).
	//
	// staticcheck is the highest-value third-party tool (dead code, misused stdlib APIs,
	// simplifications); golangci-lint bundles dozens of linters behind one pass for CI; errcheck
	// insists every error is handled. Start with gofmt + go vet + staticcheck and add more only
	// when a real bug motivates it.
}

// profiling shows live runtime settings and lists the diagnostics tools.
func profiling() {
	fmt.Println("\n--- profiling and diagnostics ---")

	// pprof - profiles: profile (CPU), heap, goroutine, allocs, block, mutex, threadcreate:
	//	go test -cpuprofile cpu.out -memprofile mem.out -bench . ./basics/014_testing
	//	go tool pprof -http=:8080 cpu.out
	// In a server, import _ "net/http/pprof" and open /debug/pprof/. The goroutine profile is the
	// fastest way to find a leak: every goroutine and exactly where it is blocked.
	//
	// Execution tracing shows scheduling, GC, syscalls and goroutine lifetimes on a timeline:
	//	go test -trace trace.out ./... && go tool trace trace.out
	// runtime/trace.NewFlightRecorder (Go 1.25) keeps a rolling in-memory window, so you can dump
	// the last few seconds WHEN something goes wrong - ideal for a rare production stall.

	// Since Go 1.25 GOMAXPROCS is container-aware: it reads the cgroup CPU limit, so a pod
	// limited to 2 CPUs on a 64-core node no longer runs 64 threads of Go code.
	fmt.Printf("GOMAXPROCS=%d NumCPU=%d NumGoroutine=%d\n",
		runtime.GOMAXPROCS(0), runtime.NumCPU(), runtime.NumGoroutine())

	// GOGC is the heap growth target (default 100: collect when the heap doubles). GOMEMLIMIT
	// (Go 1.19) is a soft memory ceiling; GOGC=off plus GOMEMLIMIT suits containers well.
	fmt.Println("current GOGC percent:", currentGCPercent())

	//	GODEBUG=gctrace=1   one line per GC cycle, on stderr
	//	GOTRACEBACK=all     every goroutine's stack on a crash
	// Prefer runtime/metrics to runtime.ReadMemStats for anything long-lived. runtime/debug has
	// ReadBuildInfo, SetGCPercent, SetMemoryLimit and PrintStack.
}

func currentGCPercent() int {
	// There is no getter, so set and restore. Each call can trigger a GC - fine for a demo; in
	// production read the "/gc/gogc:percent" sample from runtime/metrics instead.
	current := debug.SetGCPercent(100) // returns the previous value
	debug.SetGCPercent(current)
	return current
}

// projectLayout lists the conventions that matter, since there is no official layout.
func projectLayout() {
	fmt.Println("\n--- project layout and conventions ---")

	// There is NO official project layout; golang-standards/project-layout is not endorsed by the
	// Go team and is usually far more structure than warranted. The toolchain enforces only: one
	// package per directory, internal/ visibility, the _test.go suffix and filename constraints.
	//	cmd/<name>/main.go   one directory per binary, when there are several
	//	internal/            importable only from this module - compiler-enforced
	//	pkg/                 a convention for public library code; often unnecessary
	//	testdata/            ignored by the go tool; fixtures and fuzz corpora
	// START FLAT and add directories when a package has a separate reason to change.
	fmt.Println("one program per directory, as in basics/: go run ./basics/015_modules_and_tooling")

	// Name packages for what they PROVIDE, not what they contain: http, strconv, slices - never
	// util, common, helpers, base, misc. A package that could hold anything ends up holding
	// everything and becomes an import-cycle magnet (examples/common/ is a mild case).
	// Lower case, one word, no underscores, no plurals - the name repeats at every call site.
	// Avoid stutter: bytes.Buffer, not bytes.BytesBuffer; user.User at worst.
	// Interfaces belong with the CONSUMER (module 008). Keep main tiny - flags, wiring, one call,
	// error handling - because nothing in main can be imported or tested from elsewhere.
}
