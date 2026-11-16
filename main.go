// Command go-training prints a map of this Go training repository.
//
// The language course lives under basics/, one directory per module. Every module is its own
// small program: read its main() first (the table of contents), then the topic functions in
// order, and run it to compare the output with the code:
//
//	go run .                       # list every module
//	go run ./basics/005_functions  # run one module
//
// The exercise solutions listed in notes.md live under examples/: the
// examples package itself holds the CLI tools, the REST API, tic-tac-toe and
// MonetaryAmount; examples/concurrency the goroutine material;
// examples/common the shared generic helpers; and the sub-packages
// examples/chat, examples/budget and examples/db the chat, household budget
// and flat-file database exercises.
package main

import "fmt"

// modules lists the course directories under basics/, in teaching order.
var modules = []struct{ dir, title string }{
	{"001a_comments_packages_variables_scope_shadowing", "Comments, Packages, Variables, Zero Values, Scope, Shadowing"},
	{"001b_constants_iota_init", "Constants, iota, Package Variables, init"},
	{"002a_basic_types", "Basic Types: numbers, booleans, strings, runes"},
	{"002b_composite_types", "Composite Types: arrays, slices, maps, structs, defined types"},
	{"003_operators", "Operators: arithmetic, comparison, logical, bitwise, precedence"},
	{"004_control_flow", "Control Flow: if, switch, for, range, labels, goto, defer"},
	{"005_functions", "Functions: returns, variadics, values, closures, panic/recover"},
	{"006_pointers_and_memory", "Pointers and Memory: value semantics, escape analysis, aliasing, unsafe"},
	{"007_structs_and_methods", "Structs and Methods: receivers, method sets, embedding, tags"},
	{"008_interfaces", "Interfaces: implicit satisfaction, the nil trap, type switches, composition"},
	{"009_errors", "Errors: sentinels, wrapping, custom types, errors.Is/As/AsType/Join"},
	{"010_generics", "Generics: type parameters, constraints, inference, generic methods (1.27)"},
	{"011_concurrency", "Concurrency: goroutines, channels, select, sync, context, patterns"},
	{"012_iterators_and_collections", "Iterators and Collections: range-over-func, iter, slices, maps, cmp"},
	{"013_stdlib_essentials", "Standard Library: fmt, strings, time, os/io, json + json/v2, slog, uuid"},
	{"014_testing", "Testing: table-driven tests, benchmarks, fuzzing, examples, synctest (go test)"},
	{"015_modules_and_tooling", "Modules and Tooling: go.mod, build constraints, embed, vet, profiling"},
	{"016_whats_new_1_21_to_1_27", "What Changed, Go 1.21 to 1.27: the delta since you last learned Go"},
	{"017_cli_applications", "CLI Applications: flag, subcommands, validation, env vars, configuration"},
	{"018_http_services", "HTTP Services: handlers, routing, middleware, errors, graceful shutdown"},
	{"019_database_sql_and_repository", "database/sql and the Repository Pattern (docker-compose up -d)"},
	{"020_gorm", "GORM: models, AutoMigrate, CRUD, Preload and N+1, hooks (docker-compose up -d)"},
	{"021_publishing_libraries_and_cli", "Publishing Libraries and CLI Tools: API design, versioning, releases"},
}

func main() {
	fmt.Println("Go training - language modules")
	fmt.Println()
	fmt.Println("every module is its own program; run one with:")
	fmt.Println("  go run ./basics/<dir>      e.g. go run ./basics/005_functions")
	fmt.Println()
	for _, m := range modules {
		fmt.Printf("  %-50s %s\n", m.dir, m.title)
	}
	fmt.Println()
	fmt.Println("other demos in this repo (see notes.md for the exercise statements):")
	fmt.Println("  these are library packages - call them from a test or a scratch main,")
	fmt.Println("  except the chat, which is two real commands:")
	fmt.Println()
	fmt.Println("    examples              Echo(), Cat(), Find(), Grep(), TicTacToe(), Reflect(),")
	fmt.Println("                          MonetaryAmountDemo(), RestAPI() (needs docker-compose up)")
	fmt.Println("    examples/concurrency  Barbers(), ChannelsDemo(), FindFiles(pattern, roots...),")
	fmt.Println("                          ProducerConsumerClassic(), ProducerConsumerChannels(),")
	fmt.Println("                          Barrier, Semaphore")
	fmt.Println("    examples/common       Stack[T], Add[T Number], ToBytes/FromBytes (gob)")
	fmt.Println("    examples/budget       New(path), ParseEntry(args), Load/Add/Print/Save")
	fmt.Println("    examples/db           Open(path, gen), DatabaseTest(), DatabaseExercise() (HTTP on :8080)")
	fmt.Println()
	fmt.Println("    go run ./examples/chat/server    TCP chat server")
	fmt.Println("    go run ./examples/chat/client    TCP chat client")
}
