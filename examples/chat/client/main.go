package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"training.pl/go/examples/chat/common"
)

func main() {
	// Parse command line arguments
	serverAddr := flag.String("server", "localhost:8080", "Server address")
	nickname := flag.String("nick", "", "Your nickname")
	flag.Parse()

	// Validate nickname
	if *nickname == "" {
		fmt.Println("Error: Nickname is required")
		fmt.Println("Usage: ./client -nick <your_nickname> [-server <address>]")
		os.Exit(1)
	}

	// Set up logging to file. This used to live in init(); doing I/O there runs
	// before flag parsing and before main can decide anything, and the file was
	// not closed on every exit path.
	logFile, err := os.OpenFile("client.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, common.FileMode())
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
	} else {
		log.SetOutput(logFile)
	}
	closeLog := func() {
		if logFile != nil {
			logFile.Close()
		}
	}

	// Create connection
	conn := NewConnection(*nickname)

	// Create file transfer manager
	ft := NewFileTransfer(conn)

	// shutdown is THE single exit routine, shared by /quit, Ctrl-C and end of
	// input: tell the server we are leaving, tear the connection down, close the
	// log and exit. sync.Once guards against Ctrl-C arriving during /quit.
	var once sync.Once
	shutdown := func() {
		once.Do(func() {
			fmt.Println("\nShutting down...")
			if conn.IsConnected() {
				if err := conn.SendDisconnect("Client shutting down"); err != nil {
					fmt.Printf("could not send the disconnect message: %v\n", err)
				} else {
					// The message is only queued; give writePump a moment to put
					// it on the wire before the socket is closed.
					time.Sleep(100 * time.Millisecond)
				}
			}
			conn.Disconnect()
			closeLog()
			fmt.Println("Goodbye!")
			os.Exit(0)
		})
	}

	// Create UI
	ui := NewUI(conn, ft, shutdown)

	// Handle graceful shutdown on Ctrl-C / SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		shutdown()
	}()

	// Connect to server with retry
	go conn.ConnectWithBackoff(*serverAddr)

	// Wait for connection
	fmt.Printf("Connecting to %s...\n", *serverAddr)
	if err := conn.WaitForConnection(); err != nil {
		fmt.Printf("Error: %v\n", err)
		closeLog()
		os.Exit(1)
	}

	// Start UI; it returns when stdin is closed, which is treated like /quit.
	ui.Start()
	shutdown()
}
