// Package solutions holds the reference solutions for basic_exercises.md: three exercises
// for every section of basics/000_essentials.
//
// The files are named after the sections. ex01_basics.go solves exercises 1.1-1.3,
// ex08_interfaces.go solves 8.1-8.3, and so on. Every solution file has a _test.go file next
// to it with Example functions. Read them to see how a solution is used, and run them to
// check that the printed output still matches the "// Output:" comment. The 014 exercises are
// about writing tests, so their solution is ex14_testing_test.go itself.
//
//	go test ./basics/000_essentials/solutions                          # check every solution
//	go test -v -run ExampleGroupBy ./basics/000_essentials/solutions   # one example
//	go test -race ./basics/000_essentials/solutions                    # incl. the 011 solutions
//	go test -run '^$' -bench . -benchmem ./basics/000_essentials/solutions
//
// The five larger projects at the end of basic_exercises.md each have a package of their own
// next to this one: library, cache, logstats, search and monitor. Test them all at once with:
//
//	go test -race ./basics/000_essentials/solutions/...
//
// Try each exercise on your own before you read its solution.
package solutions
