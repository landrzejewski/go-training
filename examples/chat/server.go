package chat

import (
	"log"
	"net"
	"sync"
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
	messages := make(chan *message, 1000)
	mutex := &sync.RWMutex{}

	go func ()  {
		for message := range messages {
			mutex.RLock()
			for _, connection := range connections {
				if connection != message.sender {
					_, err := connection.Write(message.bytes)
					if err != nil {
						log.Println("Error sending message")
						continue
					}
				}
			}
			mutex.RUnlock()
		}
	}()

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

func handleConnection(connection net.Conn, messages chan<- *message) {
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
		messages <- &message{connection, messageBytes}
	}

}
