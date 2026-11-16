// Command 017_cli_applications is module 017 of the Go course: command-line applications.
//
// Modules 001-016 taught the language; 017-021 use it to build an application that starts, reads
// its configuration, serves requests, talks to a database and ships. Everything an operator gives
// a program arrives through three channels: arguments, environment variables and files.
//
// Every demo parses an EXPLICIT argument slice instead of os.Args. That is the technique, not a
// workaround: a main that takes its arguments as a parameter is a main you can test (testableMain).
//
//	go run ./basics/017_cli_applications
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	osArgsAndFlag()             // os.Args, FlagSet, flag spellings, where parsing stops
	customFlagsAndSubcommands() // flag.Value, flag.Func, subcommands, ErrHelp
	inputValidation()           // report every problem at once, fs.Visit, required flags
	exitCodesAndStreams()       // 0/1/2, os.Exit skips defers, stdout vs stderr
	environmentVariables()      // Getenv vs LookupEnv, typed lookups that name the variable
	configuration()             // defaults -> file -> env -> flags, redaction
	testableMain()              // run(args, stdin, stdout, stderr) error
}

// osArgsAndFlag shows the raw arguments and the standard flag package.
func osArgsAndFlag() {
	fmt.Println("--- os.Args and the flag package ---")

	// os.Args[0] is the program path (as invoked); the real arguments are os.Args[1:].
	fmt.Printf("os.Args = %q\n", os.Args)

	// flag conventions: ONE dash for every flag (-v and -verbose; --v is a synonym), no short/long
	// distinction and no combining (-abc is one flag named abc). For nested subcommands with their
	// own help, shell completion or GNU-style options reach for cobra, urfave/cli or kong.
	var cfg struct {
		Host    string
		Port    int
		Verbose bool
		Timeout time.Duration
	}
	fs := flag.NewFlagSet("demo", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // keeps the course output tidy; normally leave the default (stderr)
	// The Var forms bind straight into a config struct - declarations and config in one place.
	fs.StringVar(&cfg.Host, "host", "localhost", "host to bind")
	fs.IntVar(&cfg.Port, "port", 8080, "port to listen on")
	fs.BoolVar(&cfg.Verbose, "v", false, "verbose logging")
	fs.DurationVar(&cfg.Timeout, "timeout", 30*time.Second, "request timeout")
	// Also: Int64, Uint, Float64, TextVar (any encoding.TextUnmarshaler), Func.

	// Accepted spellings: -flag value, -flag=value, and plain -flag for booleans.
	args := []string{"-host", "example.com", "-port=9090", "-v", "-timeout", "5s", "serve", "extra"}
	if err := fs.Parse(args); err != nil {
		fmt.Println("parse failed:", err)
		return
	}
	fmt.Printf("parsed %q\n", args)
	fmt.Printf("  host=%q port=%d verbose=%t timeout=%v positional=%q\n",
		cfg.Host, cfg.Port, cfg.Verbose, cfg.Timeout, fs.Args())

	// fs.Bool and friends return a POINTER, valid only after Parse:
	//	var v bool = fs.Bool("v", false, "u") // ERROR: cannot use fs.Bool("v", false, "u") (value of type *bool) as bool value in variable declaration
	// and a FlagSet's Parse always takes the argument slice:
	//	fs.Parse() // ERROR: not enough arguments in call to fs.Parse

	for _, argv := range [][]string{
		{"build", "-v"},                         // parsing STOPS at the first non-flag argument
		{"-v", "--", "-not-a-flag", "file.txt"}, // a bare -- stops it explicitly
		{"-v", "true"},                          // a bool takes no separate value: write -v=false
	} {
		set := quietFlagSet("stop")
		v := set.Bool("v", false, "verbose")
		_ = set.Parse(argv)
		fmt.Printf("%-30q -> v=%-5t positional=%q\n", strings.Join(argv, " "), *v, set.Args())
	}

	// The help text is generated from the declarations, so it cannot drift from the code.
	// -h/-help is automatic: undefined, it prints this and (with ExitOnError) exits with status 2.
	var help strings.Builder
	fs.SetOutput(&help)
	fs.PrintDefaults()
	fmt.Print("fs.PrintDefaults():\n", help.String())
}

// quietFlagSet returns a ContinueOnError flag set that prints nothing (demo output stays tidy).
func quietFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

// customFlagsAndSubcommands shows new flag types via flag.Value and the subcommand pattern.
func customFlagsAndSubcommands() {
	fmt.Println("\n--- custom flag types and subcommands ---")

	// Any type implementing flag.Value { String() string; Set(string) error } can be a flag.
	var tags stringList
	var level logLevel = "info"
	fs := quietFlagSet("custom")
	fs.Var(&tags, "tag", "tag to apply (repeatable)")
	fs.Var(&level, "level", "log level: debug, info, warn, error")
	if err := fs.Parse([]string{"-tag", "alpha", "-tag", "beta", "-level", "debug"}); err != nil {
		fmt.Println("parse failed:", err)
	}
	fmt.Printf("repeatable flag: tags=%v  enum flag: level=%q\n", tags, level)

	// The enum rejects a bad value at PARSE time, not somewhere deep in the program.
	bad := quietFlagSet("bad")
	var badLevel logLevel
	bad.Var(&badLevel, "level", "log level")
	fmt.Println("rejected at parse time:", bad.Parse([]string{"-level", "verbose"}))

	// fs.Func: a one-off parse with no new type. (fs.TextVar takes any encoding.TextUnmarshaler,
	// so a type that already round-trips as text is a flag with no extra code.)
	pairs := map[string]string{}
	fn := quietFlagSet("fn")
	fn.Func("set", "key=value (repeatable)", func(v string) error {
		k, val, ok := strings.Cut(v, "=")
		if !ok {
			return fmt.Errorf("%q is not key=value", v)
		}
		pairs[k] = val
		return nil
	})
	_ = fn.Parse([]string{"-set", "env=prod", "-set", "region=eu"})
	fmt.Println("flag.Func collected:", pairs)

	// flag has no subcommands, but the pattern is short: switch on the first positional argument,
	// then parse the rest with that branch's OWN FlagSet. The go tool itself works this way.
	for _, argv := range [][]string{
		{"serve", "-port", "9000"},
		{"migrate", "-steps", "3", "-dry-run"},
		{"version"},
		{"explode"},
		{},
	} {
		fmt.Printf("  %-28q -> %s\n", strings.Join(argv, " "), dispatch(argv))
	}

	// Error modes of NewFlagSet: ExitOnError calls os.Exit(2) (what flag.CommandLine uses - kills
	// a test), ContinueOnError returns the error (use this: testable, main picks the exit code),
	// PanicOnError panics. -h returns flag.ErrHelp - the user got help, so that is exit 0.
	helpFs := quietFlagSet("help")
	helpFs.Bool("x", false, "something")
	fmt.Println("`-h` returns flag.ErrHelp:", errors.Is(helpFs.Parse([]string{"-h"}), flag.ErrHelp))
}

// stringList is a repeatable flag: -tag a -tag b yields []string{"a", "b"}.
type stringList []string

func (l *stringList) String() string { return strings.Join(*l, ",") }

func (l *stringList) Set(v string) error {
	if v == "" {
		return errors.New("empty value")
	}
	*l = append(*l, v) // Set is called once per occurrence, so appending gives repeatability
	return nil
}

// logLevel is an enum flag that rejects anything outside a fixed set.
type logLevel string

func (l *logLevel) String() string { return string(*l) }

func (l *logLevel) Set(v string) error {
	switch v {
	case "debug", "info", "warn", "error":
		*l = logLevel(v)
		return nil
	default:
		return fmt.Errorf("invalid log level %q (want debug, info, warn or error)", v)
	}
}

// dispatch is the whole subcommand pattern: pick the branch, then let it parse the rest.
func dispatch(argv []string) string {
	if len(argv) == 0 {
		return "usage: tool <serve|migrate|version> [flags]"
	}
	cmd, rest := argv[0], argv[1:]
	switch cmd {
	case "serve":
		fs := quietFlagSet("serve")
		port := fs.Int("port", 8080, "port to listen on")
		if err := fs.Parse(rest); err != nil {
			return "serve: " + err.Error()
		}
		return fmt.Sprintf("serve on port %d", *port)
	case "migrate":
		fs := quietFlagSet("migrate")
		steps := fs.Int("steps", 1, "how many migrations to apply")
		dry := fs.Bool("dry-run", false, "print without applying")
		if err := fs.Parse(rest); err != nil {
			return "migrate: " + err.Error()
		}
		return fmt.Sprintf("migrate %d step(s), dryRun=%t", *steps, *dry)
	case "version":
		return "tool v1.0.0"
	default:
		return fmt.Sprintf("unknown subcommand %q", cmd)
	}
}

// inputValidation shows validating parsed input, reporting every problem in one go.
func inputValidation() {
	fmt.Println("\n--- validating user input ---")

	// flag guarantees the SHAPE (an int, a duration); only your code knows the RULES.
	good := config{Host: "localhost", Port: 8080, Timeout: time.Second, Source: "in.txt"}
	fmt.Println("valid config ->", validate(good))

	// Report EVERY problem at once - not one per run, costing the user a round trip each.
	bad := config{Port: 99999, Timeout: -1, Retries: -2, Source: "a", URL: "b"}
	if err := validate(bad); err != nil {
		fmt.Println("every problem reported at once:")
		for line := range strings.SplitSeq(err.Error(), "\n") {
			fmt.Println("  " + line)
		}
	}

	// Was a flag actually SET? A default looks exactly like an explicitly given equal value.
	// fs.Visit walks only the flags the user set, fs.VisitAll walks all of them.
	fs := quietFlagSet("visit")
	fs.String("host", "localhost", "host")
	fs.Int("port", 8080, "port")
	fs.Bool("v", false, "verbose")
	_ = fs.Parse([]string{"-port", "8080"}) // the SAME value as the default, but set explicitly
	var set, all []string
	fs.Visit(func(f *flag.Flag) { set = append(set, f.Name) })
	fs.VisitAll(func(f *flag.Flag) { all = append(all, f.Name) })
	fmt.Printf("fs.Visit (set): %v  fs.VisitAll (all): %v\n", set, all)

	// That is how required flags (and configuration precedence, see configuration) work.
	fmt.Println("required check:", requireFlags(fs, "host", "port"))

	// Environmental rules (file exists, port free) can change between check and use - they are
	// checks, not guarantees. Never trust a user path: confine access with os.Root (module 013),
	// filepath.Clean plus a prefix check is famously easy to defeat. Report errors on stderr.
}

type config struct {
	Host    string
	Port    int
	Timeout time.Duration
	Retries int
	Source  string // exactly one of Source/URL
	URL     string
}

// validate collects every failure with errors.Join (module 009): one failure per line.
func validate(c config) error {
	var problems []error

	// 1. per-field rules: ranges, patterns, non-empty strings
	if c.Host == "" {
		problems = append(problems, errors.New("host must not be empty"))
	}
	if c.Port < 1 || c.Port > 65535 {
		problems = append(problems, fmt.Errorf("port %d is out of range 1..65535", c.Port))
	}
	if c.Timeout <= 0 {
		problems = append(problems, fmt.Errorf("timeout must be positive, got %v", c.Timeout))
	}
	if c.Retries < 0 {
		problems = append(problems, fmt.Errorf("retries must not be negative, got %d", c.Retries))
	}

	// 2. cross-field rules: ordering, mutually exclusive or dependent flags
	switch {
	case c.Source == "" && c.URL == "":
		problems = append(problems, errors.New("one of -source or -url is required"))
	case c.Source != "" && c.URL != "":
		problems = append(problems, errors.New("-source and -url are mutually exclusive"))
	}

	return errors.Join(problems...) // nil when problems is empty
}

// requireFlags reports which of the named flags the user did not set.
func requireFlags(fs *flag.FlagSet, names ...string) error {
	provided := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { provided[f.Name] = true })
	var missing []error
	for _, n := range names {
		if !provided[n] {
			missing = append(missing, fmt.Errorf("-%s is required", n))
		}
	}
	return errors.Join(missing...)
}

// exitCodesAndStreams shows exit statuses, why os.Exit is dangerous, and stdout vs stderr.
func exitCodesAndStreams() {
	fmt.Println("\n--- exit codes, stdout vs stderr, os.Exit ---")

	// Shells, CI, make and && all read the exit status. Conventions: 0 success, 1 general
	// failure, 2 usage error (what flag uses). Returning from main is 0; an unrecovered panic
	// exits with 2 and prints a stack trace to stderr.

	// os.Exit runs NO deferred functions - no file closed, no lock released, no recover.
	// log.Fatal calls os.Exit(1), so it has the same problem; log.Panic at least unwinds.
	fmt.Println(deferWithReturn())
	fmt.Println("  (with os.Exit instead of return, that cleanup would never run)")

	// Rule: call os.Exit in exactly ONE place, at the bottom of main, never in a library.
	// The work lives in run() error, where defer works (see testableMain):
	//	func main() {
	//		if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
	//			fmt.Fprintln(os.Stderr, "myapp:", err)
	//			os.Exit(exitStatus(err))
	//		}
	//	}

	// Map the error to a status in one place.
	for _, err := range []error{
		nil,
		errors.New("disk full"),
		fmt.Errorf("bad usage: %w", flag.ErrHelp),
		usageError{errors.New("-port is required")},
	} {
		fmt.Printf("exit status for %-32v -> %d\n", err, exitStatus(err))
	}

	// stdout carries DATA, stderr carries DIAGNOSTICS (errors, progress, logs), so that
	// `mytool | jq` receives only the data. Prefix messages with the program name.
	fmt.Fprintln(os.Stderr, "017_cli_applications: this line went to stderr")
}

func deferWithReturn() (result string) {
	defer func() { result += " -> and this deferred cleanup ran" }()
	return "returned normally"
}

// usageError marks an error as a usage problem, which maps to exit status 2.
type usageError struct{ Err error }

func (e usageError) Error() string { return e.Err.Error() }
func (e usageError) Unwrap() error { return e.Err }

// exitStatus turns an error into a process exit status.
func exitStatus(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, flag.ErrHelp):
		return 0 // the user asked for help and got it: that is success
	case errors.As(err, new(usageError)):
		return 2
	default:
		return 1
	}
}

// environmentVariables shows reading the environment, and typed lookups that fail usefully.
func environmentVariables() {
	fmt.Println("\n--- environment variables ---")

	// os.Getenv returns "" for an UNSET variable - indistinguishable from one set to "".
	// os.LookupEnv returns (value, ok); for configuration that difference nearly always matters.
	const unset = "MYAPP_DEFINITELY_NOT_SET"
	fmt.Printf("os.Getenv(%q) = %q\n", unset, os.Getenv(unset))
	v, ok := os.LookupEnv(unset)
	fmt.Printf("os.LookupEnv(%q) = %q, ok=%t\n", unset, v, ok)

	// Conventions (12-factor): SCREAMING_SNAKE_CASE, prefixed with the app name (MYAPP_PORT).
	// In tests use t.Setenv (module 014): it restores the value and forbids t.Parallel, because
	// the environment is process-wide state.
	vars := map[string]string{
		"MYAPP_EMPTY": "", "MYAPP_PORT": "9090", "MYAPP_DEBUG": "true",
		"MYAPP_TIMEOUT": "15s", "MYAPP_BROKEN": "not-a-number",
	}
	for k, val := range vars {
		_ = os.Setenv(k, val)
	}
	defer func() {
		for k := range vars {
			_ = os.Unsetenv(k)
		}
	}()
	v, ok = os.LookupEnv("MYAPP_EMPTY")
	fmt.Printf("set to the empty string: value=%q ok=%t  <- ok is TRUE\n", v, ok)

	// Everything arrives as a string, so every conversion can fail - name the VARIABLE in the
	// error: "MYAPP_PORT: invalid syntax" is actionable, "invalid syntax" is not.
	port, err := envValue("MYAPP_PORT", 8080, strconv.Atoi)
	fmt.Printf("MYAPP_PORT    -> %d (err=%v)\n", port, err)
	debug, err := envValue("MYAPP_DEBUG", false, strconv.ParseBool)
	fmt.Printf("MYAPP_DEBUG   -> %t (err=%v)\n", debug, err)
	timeout, err := envValue("MYAPP_TIMEOUT", time.Minute, time.ParseDuration)
	fmt.Printf("MYAPP_TIMEOUT -> %v (err=%v)\n", timeout, err)
	broken, err := envValue("MYAPP_BROKEN", 1, strconv.Atoi)
	fmt.Printf("MYAPP_BROKEN  -> %d (err=%v)\n", broken, err)
	fallback, err := envValue("MYAPP_ABSENT", 42, strconv.Atoi) // unset is a default, not an error
	fmt.Printf("MYAPP_ABSENT  -> %d (err=%v)\n", fallback, err)

	// Also: os.Unsetenv, os.Environ() (all as KEY=value), os.ExpandEnv ($VAR substitution).
	fmt.Printf("os.ExpandEnv: %q, %d variables in os.Environ()\n",
		os.ExpandEnv("port is $MYAPP_PORT"), len(os.Environ()))

	// The environment is inherited by every child process and shows up in crash dumps and
	// (on some systems) `ps e`. Secrets belong in a file or a secret manager - not in a
	// -password flag and preferably not in the environment.
}

// envValue reads key and converts it with parse (a generic function, module 010).
func envValue[T any](key string, fallback T, parse func(string) (T, error)) (T, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback, nil
	}
	v, err := parse(raw)
	if err != nil {
		return fallback, fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}

// configuration shows combining four configuration sources with a predictable precedence.
func configuration() {
	fmt.Println("\n--- application configuration ---")

	// Precedence, lowest to highest - the more specific and immediate the source, the higher:
	//	1. compiled-in defaults   the program runs with no configuration at all
	//	2. a configuration file   the deployment's settings, version-controlled
	//	3. environment variables  the container's or operator's overrides
	//	4. command-line flags     this one invocation
	const configFile = `{"host": "file-host", "port": 7070, "logLevel": "warn"}`
	env := map[string]string{"MYAPP_HOST": "env-host", "MYAPP_PORT": "9090"}
	args := []string{"-host", "flag-host"}

	for _, s := range []struct {
		label string
		file  string
		env   map[string]string
		args  []string
	}{
		{"defaults only", "", nil, nil},
		{"+ config file", configFile, nil, nil},
		{"+ environment", configFile, env, nil},
		{"+ flags", configFile, env, args},
	} {
		cfg, err := loadConfig(s.file, s.env, s.args)
		if err != nil {
			fmt.Printf("%-14s -> error: %v\n", s.label, err)
			continue
		}
		fmt.Printf("%-14s -> %s\n", s.label, cfg)
	}
	fmt.Println("host from the flag, port from env, logLevel from the file, timeout from the default")

	// An unknown key in the file is an error, not a silently ignored typo.
	if _, err := loadConfig(`{"hostt": "typo"}`, nil, nil); err != nil {
		fmt.Println("DisallowUnknownFields catches a typo:", err)
	}

	// Offer a -config-dump that prints the EFFECTIVE configuration with secrets redacted:
	// "why is it connecting to the wrong host" becomes a five-second question.
	withSecret := defaultConfig()
	withSecret.Password = "hunter2"
	fmt.Println("String() redacts:", withSecret)

	// Load into ONE struct, validate it ONCE, pass it explicitly - a global Config is convenient
	// for a week and untestable forever. viper/koanf do all this; for a few settings it is ~40
	// lines and no dependency (JSON needs none either; TOML/YAML do).
}

// appConfig is loaded from four layers, then validated once.
type appConfig struct {
	Host     string        `json:"host"`
	Port     int           `json:"port"`
	Timeout  time.Duration `json:"-"` // durations need custom JSON handling (module 013)
	LogLevel string        `json:"logLevel"`
	Password string        `json:"password"`
}

// String redacts the secret, so the struct is safe to print.
func (c appConfig) String() string {
	password := "(unset)"
	if c.Password != "" {
		password = "***redacted***"
	}
	return fmt.Sprintf("host=%s port=%d timeout=%v logLevel=%s password=%s",
		c.Host, c.Port, c.Timeout, c.LogLevel, password)
}

// defaultConfig is layer 1, kept in code - not in a sample file the user might not have.
func defaultConfig() appConfig {
	return appConfig{Host: "localhost", Port: 8080, Timeout: 30 * time.Second, LogLevel: "info"}
}

// loadConfig applies all four layers in order, lowest precedence first.
func loadConfig(fileJSON string, env map[string]string, args []string) (appConfig, error) {
	cfg := defaultConfig() // 1. defaults

	if fileJSON != "" { // 2. configuration file
		dec := json.NewDecoder(strings.NewReader(fileJSON))
		dec.DisallowUnknownFields() // a typo is an error, not a silently ignored key
		if err := dec.Decode(&cfg); err != nil {
			return cfg, fmt.Errorf("config file: %w", err)
		}
	}

	if v := env["MYAPP_HOST"]; v != "" { // 3. environment
		cfg.Host = v
	}
	if v := env["MYAPP_PORT"]; v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return cfg, fmt.Errorf("MYAPP_PORT: %w", err)
		}
		cfg.Port = n
	}
	if v := env["MYAPP_LOG_LEVEL"]; v != "" {
		cfg.LogLevel = v
	}

	// 4. flags, applied with fs.Visit: only flags the user actually SET override anything.
	// Reading every flag unconditionally works only while each default is the already-resolved
	// value; with a literal default (8080) it would silently overwrite the environment.
	fs := quietFlagSet("app")
	host := fs.String("host", cfg.Host, "host to bind")
	port := fs.Int("port", cfg.Port, "port to listen on")
	level := fs.String("log-level", cfg.LogLevel, "log level")
	timeout := fs.Duration("timeout", cfg.Timeout, "request timeout")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "host":
			cfg.Host = *host
		case "port":
			cfg.Port = *port
		case "log-level":
			cfg.LogLevel = *level
		case "timeout":
			cfg.Timeout = *timeout
		}
	})
	return cfg, nil
}

// testableMain shows the run() pattern that makes a whole program testable.
func testableMain() {
	fmt.Println("\n--- a testable main ---")

	// func main() takes nothing, returns nothing and touches global state (os.Args, env,
	// os.Stdin, os.Stdout, os.Exit) - untestable. So main only wires the real world in and
	// picks the exit status; run is an ordinary function. The same goes for everything else
	// main touches: env as a map parameter, the clock as a func() time.Time.
	// A test does exactly this - pass arguments and buffers, assert (table-driven, module 014):
	for _, c := range []struct {
		args  []string
		stdin string
	}{
		{[]string{"-name", "Ada"}, ""},
		{[]string{"-name", "Ada", "-upper"}, ""},
		{nil, "Grace\n"},
		{[]string{"-nope"}, ""},
		{nil, ""},
	} {
		var stdout, stderr strings.Builder
		err := run(c.args, strings.NewReader(c.stdin), &stdout, &stderr)
		label := fmt.Sprintf("args=%v stdin=%q", c.args, c.stdin)
		if err != nil {
			fmt.Printf("%-33s -> exit %d, error: %v\n", label, exitStatus(err), err)
		} else {
			fmt.Printf("%-33s -> exit 0, stdout: %q\n", label, strings.TrimSpace(stdout.String()))
		}
	}
	// No subprocess, no capturing os.Stdout, no golden files. The CLI exercises in notes.md
	// (echo, cat, find, grep - solved in examples/) are the place to practise this shape.
}

// run is the whole application as an ordinary function: args, stdin and outputs are parameters.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("greet", flag.ContinueOnError)
	fs.SetOutput(stderr)
	upper := fs.Bool("upper", false, "upper-case the greeting")
	name := fs.String("name", "", "name to greet; if empty, read from stdin")
	if err := fs.Parse(args); err != nil {
		return usageError{err}
	}

	who := *name
	if who == "" {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		who = strings.TrimSpace(string(data))
	}
	if who == "" {
		return usageError{errors.New("-name is required, or provide a name on stdin")}
	}

	greeting := "Hello, " + who + "!"
	if *upper {
		greeting = strings.ToUpper(greeting)
	}
	fmt.Fprintln(stdout, greeting)
	return nil
}
