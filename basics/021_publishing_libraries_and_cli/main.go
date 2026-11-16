// Command 021_publishing_libraries_and_cli is module 021 of the Go course: publishing libraries
// and CLI tools - the last module.
//
// Go has NO registry and NO publish command: a module is identified by its own URL and
// "publishing" is git tag + git push; the module proxy discovers it from there. Nobody can take
// your name or delete your release - and nobody can fix your mistake either.
//
//	go run ./basics/021_publishing_libraries_and_cli
//	go run -ldflags="-X main.version=1.2.3" ./basics/021_publishing_libraries_and_cli
package main

import (
	"fmt"
	"go/doc/comment"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strings"
)

func main() {
	whatIsPublishable() // the module path is a URL; LICENSE, README, internal/
	apiDesign()         // every exported name is a promise; functional options
	documentation()     // doc comments are rendered by pkg.go.dev; Example functions
	versioning()        // semver, the /v2 import-path rule, what counts as breaking
	publishing()        // git tag, the proxy, immutable releases, retract, GOPRIVATE
	shippingCLI()       // static binaries, cross-compiling, -ldflags -X, ReadBuildInfo
	releaseChecklist()  // everything above, in the order you apply it
}

// whatIsPublishable shows what identifies a module and what it needs before the first tag.
func whatIsPublishable() {
	fmt.Println("--- what makes a module publishable ---")

	// `module github.com/you/thing` means the code lives at that address - that is how go get
	// finds it. So the path must be FETCHABLE (a public repo, or a private one plus GOPRIVATE).
	if info, ok := debug.ReadBuildInfo(); ok {
		fmt.Printf("this module: path=%s version=%s\n", info.Main.Path, orDefault(info.Main.Version, "(devel)"))
	}
	fmt.Println("training.pl/go resolves to no repository, so it could not be published as is")

	// The path IS the identity: renaming the repository breaks every importer. A vanity path
	// (go.uber.org/zap) works by serving <meta name="go-import"> pointing at the real repo - the
	// only indirection the toolchain offers. A module in a subdirectory is imported as
	// github.com/you/repo/sub and tagged sub/v1.2.3 - possible, but root = module is simpler.
	//
	// Before the first tag: the FINAL module path (changing it later is breaking), a LICENSE
	// (without one it is "all rights reserved" and pkg.go.dev warns), a README (pkg.go.dev shows
	// it) and a committed go.sum.
	//
	// internal/ is unimportable from outside the module (module 015), so it is not API. Start
	// with almost everything internal and promote deliberately: you can always export more,
	// never less.
}

func orDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// apiDesign shows the forward-compatible shape of a constructor.
func apiDesign() {
	fmt.Println("\n--- designing an API you can live with ---")

	// Required arguments positional, everything optional an Option (module 005). Adding a
	// parameter breaks every caller; adding an Option breaks nobody:
	//	func New(host string, port int, timeout time.Duration) *Client // adding a parameter breaks
	//	func New(host string, opts ...Option) *Client                  // adding an option does not
	fmt.Println("defaults:    ", newClient("api.example.com"))
	fmt.Println("with options:", newClient("api.example.com", withPort(8443), withRetries(5), withVerbose()))

	// The rules this course already taught, applied knowing you cannot take them back:
	//   - minimise the surface: 4 exported functions can be reworked, 40 cannot
	//   - accept interfaces, return structs (module 008): a struct can gain methods, a returned
	//     interface freezes its method set forever
	//   - make the zero value useful (module 001a): var b bytes.Buffer just works
	//   - document which errors you return (module 009): callers errors.Is against them -> API
	//   - context.Context first in anything that blocks (module 011): adding it later breaks
	//   - a `_ struct{}` field, or keeping a struct uncomparable (module 007), stops callers
	//     depending on positional literals or ==
	//   - no exported mutable package variables: global state you can never reclaim
}

// client returns a *struct* from its constructor, not an interface, so methods can be added
// without breaking anyone.
type client struct {
	host    string
	port    int
	retries int
	verbose bool
}

type option func(*client)

func withPort(p int) option    { return func(c *client) { c.port = p } }
func withRetries(n int) option { return func(c *client) { c.retries = n } }
func withVerbose() option      { return func(c *client) { c.verbose = true } }

func newClient(host string, opts ...option) *client {
	c := &client{host: host, port: 443, retries: 3} // the defaults live here
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *client) String() string {
	return fmt.Sprintf("host=%s port=%d retries=%d verbose=%t", c.host, c.port, c.retries, c.verbose)
}

// documentation shows that a doc comment is structured text, rendered by the go/doc/comment rules.
func documentation() {
	fmt.Println("\n--- documentation is part of the API ---")

	// pkg.go.dev renders doc comments automatically - for most users that page IS your library.
	// The rules are module 001a's: the comment sits right above the declaration and starts with
	// its name; the package comment (conventionally in doc.go) opens the page - say what the
	// package is for and show the smallest complete usage. Since Go 1.19 doc comments have
	// # headings, lists, indented code and [Name] links; a Deprecated: paragraph is surfaced by
	// editors and linters. This is exactly how gofmt, go doc and pkg.go.dev parse one:
	const doc = `Dial connects to the server.

# Errors

It returns [ErrTimeout] when the server does not answer, and otherwise
an error wrapping the [net.OpError]:
  - check timeouts with errors.Is
  - retry other errors at most once

Deprecated: use NewClient instead. Dial will be removed in v3.
`
	p := comment.Parser{LookupSym: func(recv, name string) bool { return name == "ErrTimeout" }}
	pr := comment.Printer{DocLinkBaseURL: "https://pkg.go.dev", HeadingID: func(*comment.Heading) string { return "" }}
	fmt.Println("rendered as Markdown:")
	for line := range strings.SplitSeq(strings.TrimSpace(string(pr.Markdown(p.Parse(doc)))), "\n") {
		fmt.Println(strings.TrimRight("  "+line, " "))
	}

	// Read your own docs BEFORE tagging - it catches the paragraph that made sense only to you:
	//	go doc -u ./basics/021_publishing_libraries_and_cli version   (terminal; -u: unexported too)
	//	go doc -http=:6060                                            (Go 1.25: the whole site, offline)
	//
	// Example functions are the best documentation you can write (module 014 has
	// Example_reverse, Example_parseKV, Example_parseKVUnordered in basics/014_testing/main_test.go):
	// go test RUNS them, so the // Output: comment cannot go stale; pkg.go.dev shows them with a
	// Run button; in package x_test they can only use the EXPORTED API - so a clumsy example is
	// telling you the API itself is clumsy.
}

// versioning shows semantic import versioning: from v2 on, the major version is in the path.
func versioning() {
	fmt.Println("\n--- semantic versioning and the /v2 rule ---")

	// v0.x.y   anything may change; the toolchain treats every v0 minor bump as breaking.
	//          The honest place while an API still moves - do not rush to v1.
	// v1.x.y   stable: patch = fixes, minor = additive only, never breaking
	// v2.0.0+  breaking - and the IMPORT PATH must change: go.mod says
	//          `module github.com/you/thing/v2`, every importer writes that path, but the git
	//          tag keeps the plain form v2.0.0. The point: v1 and v2 COEXIST in one build, so a
	//          big dependency graph migrates one package at a time. The standard library does
	//          the same: math/rand/v2, encoding/json/v2.
	// Lay out v2 either in a v2/ subdirectory (both stay buildable on one branch) or on a v2
	// branch with the path updated in go.mod.
	for _, v := range []string{"v0.9.0", "v1.4.2", "v2.0.0", "v3.1.0"} {
		fmt.Printf("tag %s -> import %q\n", v, importPath("github.com/you/thing", v))
	}

	// What counts as BREAKING is broader than people expect:
	for _, change := range []struct{ what, breaking string }{
		{"removing or renaming an exported name", "yes"},
		{"changing a function signature", "yes"},
		{"adding a method to an exported INTERFACE", "yes - implementers break"},
		{"adding a method to an exported STRUCT", "no"},
		{"adding a field to an exported struct", "yes, if callers use positional literals"},
		{"adding a new exported function", "no"},
		{"changing documented behaviour", "yes, even with the same signature"},
		{"adding a variadic Option parameter", "no - that is the whole point"},
	} {
		fmt.Printf("  %-42s %s\n", change.what, change.breaking)
	}
}

// importPath returns the path importers must use for a module path and a release tag.
func importPath(modulePath, tag string) string {
	major, _, _ := strings.Cut(strings.TrimPrefix(tag, "v"), ".")
	if major == "0" || major == "1" {
		return modulePath // v0 and v1 have no suffix
	}
	return modulePath + "/v" + major
}

// publishing shows how a release reaches users - and why it can never be taken back.
func publishing() {
	fmt.Println("\n--- publishing, the proxy, and retract ---")

	// Publishing is two commands:
	//	git tag v1.2.3
	//	git push origin v1.2.3
	// The first `go get github.com/you/thing@v1.2.3` makes the proxy fetch and cache it;
	// pkg.go.dev indexes it shortly after. These settings decide where go get looks:
	out, err := exec.Command("go", "env", "GOPROXY", "GOSUMDB", "GOPRIVATE").Output()
	if err != nil {
		fmt.Println("go env failed:", err)
	} else {
		vals := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
		for i, k := range []string{"GOPROXY", "GOSUMDB", "GOPRIVATE"} {
			if i < len(vals) {
				fmt.Printf("%-9s = %q\n", k, vals[i])
			}
		}
	}
	// GOPROXY defaults to proxy.golang.org with `direct` as the fallback.
	// GOPRIVATE=github.com/mycorp (glob patterns, matching path prefixes) skips the proxy AND
	// the checksum database for those paths, fetching straight from the VCS; GONOPROXY and
	// GONOSUMDB split the two behaviours.

	// RELEASES ARE IMMUTABLE. The proxy caches a version permanently and sum.golang.org records
	// its hash forever, so deleting a tag does NOT un-publish it, MOVING a tag produces a checksum
	// mismatch (SECURITY ERROR, builds break loudly), and a secret in a tagged commit is public
	// forever - rotate it, do not try to erase it.
	// The only fix for a bad release is a NEW version plus retract, in that NEWER release's go.mod
	// (retracting v1.2.3 means publishing v1.2.4); the toolchain then avoids those versions and
	// warns anyone pinned to them. Nothing removes a version:
	//	retract (
	//		v1.2.3           // contains a data-loss bug
	//		[v1.0.0, v1.1.0] // published from the wrong branch
	//	)
	//
	// Verify what you are about to depend on: go mod verify (checksums match), go list -m
	// -versions M (every published version), go mod why -m M (why it is in the graph).
}

// version is stamped at link time:
//
//	go run -ldflags="-X main.version=1.2.3" ./basics/021_publishing_libraries_and_cli
//
// -X sets a package-level STRING variable (not a constant); add -s -w to strip the symbol table
// and DWARF data, usually a quarter of the binary.
var version = "dev"

// shippingCLI shows what makes a Go binary easy to ship, and what it knows about itself.
func shippingCLI() {
	fmt.Println("\n--- shipping a CLI tool ---")

	// A Go binary is statically linked with no runtime dependency (no interpreter, no VM) -
	// Go's biggest practical advantage for command-line tools. Layout:
	//	cmd/mytool/main.go   the binary, kept tiny (module 017's run() pattern)
	//	internal/...         everything that is not API
	//	*.go                 the importable library, if any
	// `go install github.com/you/thing/cmd/mytool@latest` builds from source into $GOBIN: no
	// package manager, no root, no install script.

	// Cross-compiling needs no extra toolchain - the standard library is compiled on demand:
	//	GOOS=linux   GOARCH=amd64 go build -o dist/mytool-linux-amd64  ./cmd/mytool
	//	GOOS=windows GOARCH=amd64 go build -o dist/mytool.exe          ./cmd/mytool
	// CGO_ENABLED=0 guarantees a fully static binary (FROM scratch images, another libc).
	// `go tool dist list` prints every supported pair.
	fmt.Printf("running on %s/%s, version=%q\n", runtime.GOOS, runtime.GOARCH, version)

	// For a module built with go build/go install you often need no -X at all: the toolchain
	// stamps the module version, the VCS revision and whether the tree was dirty (Go 1.18+).
	if info, ok := debug.ReadBuildInfo(); ok {
		fmt.Printf("debug.ReadBuildInfo: module %s %s, built by %s\n",
			info.Main.Path, orDefault(info.Main.Version, "(devel)"), info.GoVersion)
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs", "vcs.revision", "vcs.time", "vcs.modified", "GOOS", "GOARCH", "-ldflags", "CGO_ENABLED":
				value := s.Value
				if s.Key == "vcs.revision" && len(value) > 12 {
					value = value[:12]
				}
				fmt.Printf("  %-12s %s\n", s.Key, value)
			}
		}
	}
	// `go run` omits the vcs.* stamps; `go build` records them.

	// GoReleaser automates the rest - every target, archives, checksums, Homebrew formulae,
	// Docker images and a GitHub release from one .goreleaser.yaml.
}

// releaseChecklist is everything above, in the order you would apply it before tagging.
func releaseChecklist() {
	fmt.Println("\n--- a release checklist ---")

	for _, g := range []struct {
		name  string
		items [][2]string
	}{
		{"CORRECTNESS", [][2]string{
			{"gofmt -l .", "formatted (module 015)"},
			{"go vet ./...", "no reported mistakes (module 015)"},
			{"staticcheck ./...", "no reported mistakes (module 015)"},
			{"go test ./...", "green (module 014)"},
			{"go test -race ./...", "green under the race detector (module 011)"},
			{"go test -count=1 ./...", "green with the cache defeated (module 014)"},
		}},
		{"API", [][2]string{
			{"go doc -http=:6060", "read your own documentation"},
			{"every exported name documented", "starting with its own name (module 001a)"},
			{"Example functions", "for the main entry points (module 014)"},
			{"nothing exported that could be internal/", "(module 015)"},
			{"optional parameters are Options", "not positional (module 005)"},
		}},
		{"MODULE", [][2]string{
			{"go mod tidy", "no unused or missing requires (module 015)"},
			{"go.sum committed", "builds are verifiable"},
			{"LICENSE and README present", ""},
			{"module path final, /vN if major >= 2", ""},
			{"retract any earlier bad release", ""},
		}},
		{"RELEASE", [][2]string{
			{"git tag vX.Y.Z && git push origin vX.Y.Z", ""},
			{"go install <path>@vX.Y.Z", "verify it installs cleanly"},
		}},
	} {
		fmt.Println(g.name)
		for _, item := range g.items {
			fmt.Println(strings.TrimRight(fmt.Sprintf("  %-42s %s", item[0], item[1]), " "))
		}
	}

	// The two rules that outlast every checklist.
	fmt.Println("\n1. releases are immutable - think before you tag, and fix FORWARD")
	fmt.Println("2. every exported identifier is a promise - the smallest API you can get away")
	fmt.Println("   with is the one you will be happiest with in two years")

	fmt.Println("\n" + strings.Repeat("-", 78))
	fmt.Println("That is the end of the course: 23 modules, from `package main` to a tagged release.")
	fmt.Println("Run `go run .` for the index, and see notes.md for the exercises.")
}
