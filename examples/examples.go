// Package examples holds the solutions to the standalone exercises listed in
// notes.md: the command-line tools (Echo, Cat, Find, Grep), the tic-tac-toe
// game (TicTacToe), the MonetaryAmount value type, a small reflection demo
// (Reflect) and a REST API backed by PostgreSQL (RestAPI).
//
// The larger exercises live in sub-packages: budget (household budget with
// JSON persistence), db (a flat-file binary database), chat (a TCP chat
// server and client), concurrency (goroutine and channel demos) and common
// (shared generic helpers).
//
// Every entry point is a plain function: call it from a scratch main or a
// test. The CLI tools read their arguments from os.Args[1:].
package examples
