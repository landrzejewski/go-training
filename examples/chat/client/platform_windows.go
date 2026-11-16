//go:build windows

package main

import (
	"os"
	"os/exec"
)

// clearScreen clears the terminal screen on Windows
func clearScreen() {
	cmd := exec.Command("cmd", "/c", "cls")
	cmd.Stdout = os.Stdout
	cmd.Run()
}
