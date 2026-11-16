//go:build !windows

package main

import "fmt"

// clearScreen clears the terminal screen on Unix-like systems
func clearScreen() {
	// ANSI escape codes work on Unix-like systems
	fmt.Print("\033[H\033[2J")
}
