package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
)

const serverAddress = "localhost:9000"

func main() {
	server()
}

func client() {
	connection, err := net.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Println("Error connecting on:", serverAddress)
	}
	defer connection.Close()

	go func() {
		for {
			message, err := bufio.NewReader(connection).ReadString('\n')
			if err != nil {
				fmt.Println("Disconnected form:", serverAddress)
				return
			}
			fmt.Print("Message: ", message)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()
		_, err := fmt.Fprintf(connection, "%s\n", input)
		if err != nil {
			fmt.Println("Error sending message:", serverAddress)
			return
		}
	}
}

func server() {
	listener, err := net.Listen("tcp", serverAddress)
	if err != nil {
		fmt.Println("Error listening on:", serverAddress)
	}
	defer listener.Close()
	fmt.Println("Listening on:", serverAddress)

	broadcast := make(chan string)
	connections := make([]net.Conn, 0)
	mutex := sync.Mutex{}

	go func() {
		for {
			message := <-broadcast
			mutex.Lock()
			for _, connection := range connections {
				_, err := fmt.Fprintf(connection, "%s", message)
				if err != nil {
					fmt.Println("Error sending:", err)
				}
			}
			mutex.Unlock()
		}
	}()

	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err.Error())
			os.Exit(1)
		}
		fmt.Println("Client connected:", connection.LocalAddr())
		mutex.Lock()
		connections = append(connections, connection)
		mutex.Unlock()
		go handleConnection(connection, broadcast)
	}
}

func handleConnection(connection net.Conn, broadcast chan<- string) {
	defer connection.Close()
	buffer := make([]byte, 1024)
	for {
		size, err := connection.Read(buffer)
		if err != nil {
			fmt.Println("Error reading from connection:", err.Error())
			os.Exit(1)
		}
		message := string(buffer[:size])
		broadcast <- message
	}
}
