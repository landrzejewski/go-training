package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
	"training.pl/examples/utils"
)

const serverAddress = "localhost:9000"
const bufferSize = 20

type message struct {
	sender net.Conn
	bytes  []byte
}

func main() {
	if len(os.Args) > 1 {
		client()
	} else {
		server()
	}
}

func client() {
	connection, err := net.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Println("Connecting to server failed: ", err)
	}
	defer func(connection net.Conn) {
		err := connection.Close()
		if err != nil {
			fmt.Println("Closing connection failed: ", err)
		}
	}(connection)

	go func() {
		for {
			buffer := make([]byte, bufferSize)
			_, err := bufio.NewReader(connection).Read(buffer)
			if err != nil {
				fmt.Println("Disconnected form: ", serverAddress)
				return
			}
			var message string
			err = utils.FromBytes(buffer, &message)
			if err != nil {
				fmt.Println("Reading message failed: ", err)
				return
			}
			fmt.Print(message + "\n")
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		buffer := make([]byte, bufferSize)
		data, _ := utils.ToBytes(scanner.Text())
		if len(data) > bufferSize {
			fmt.Println("Message too long")
			continue
		}
		copy(buffer, data[:])
		_, err := connection.Write(buffer)
		if err != nil {
			fmt.Println("Error sending message:", serverAddress)
			return
		}
	}
}

func server() {
	listener, err := net.Listen("tcp", serverAddress)
	if err != nil {
		fmt.Println("Listening failed: ", err)
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println("Closing server failed: ", err)
		}
	}(listener)

	fmt.Println("Listening on:", serverAddress)

	messages := make(chan *message)
	connections := make([]net.Conn, 0)
	mutex := sync.RWMutex{}

	go func() {
		for {
			message := <-messages
			mutex.RLock()
			for _, connection := range connections {
				if connection != message.sender {
					_, err := connection.Write(message.bytes)
					if err != nil {
						fmt.Println("Sending message failed: ", err)
					}
				}
			}
			mutex.RUnlock()
		}
	}()

	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("Accepting connection failed: ", err)
			continue
		}
		fmt.Println("Client connected:", connection.LocalAddr())
		mutex.Lock()
		connections = append(connections, connection)
		mutex.Unlock()
		go handleConnection(connection, messages)
	}
}

func handleConnection(connection net.Conn, messages chan<- *message) {
	defer func(connection net.Conn) {
		err := connection.Close()
		if err != nil {
			fmt.Println("Closing connection failed: ", err)
		}
	}(connection)

	buffer := make([]byte, bufferSize)
	for {
		_, err := connection.Read(buffer)
		if err != nil {
			fmt.Println("Reading from connection failed: ", err.Error())
			break
		}
		messages <- &message{connection, buffer}
	}
}
