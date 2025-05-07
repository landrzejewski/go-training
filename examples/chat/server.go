package chat

import (
	"fmt"
	"log"
	"net"
	"sync"

	"training.pl/go/common"
)

func Server(address string) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}
	defer func ()  {
		err := listener.Close()
		if err != nil {
			panic(err)
		}
	}()
	
	connections := make([]net.Conn, 0)
	messages := make(chan *Message, 1000)
	mutex := &sync.Mutex{}

	log.Println("Listening on: " + address)

	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Println("Connection accept error: " + err.Error())
			continue
		}
		log.Println("Client connected: ", connection.LocalAddr())
		mutex.Lock()
		connections = append(connections, connection)
		go handleConnection(connection, messages)
		mutex.Unlock()
	}

	// close(messages)
}

func handleConnection(connection net.Conn, messages chan<- *Message) {
	defer func ()  {
		err := connection.Close()
		if err != nil {
			log.Println("Error closing connection: " + err.Error())
		}
	}()
	messageBytes := make([]byte, bufferSize)
	for {
		_, err := connection.Read(messageBytes)
		if err != nil {
			log.Println("Error reading message")
			break
		}
		var text string
		err = common.FromBytes(messageBytes, &text)
		fmt.Println(text)
	}


}