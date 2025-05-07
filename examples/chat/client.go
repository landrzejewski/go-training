package chat

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"

	"training.pl/go/common"
)

func Client(address string) {
	connection , err := net.Dial("tcp", address)
	if err != nil {
		panic(err)
	}

	go listenForMessages(connection)

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		text := scanner.Text()
		textBytes, err := common.ToBytes(text)
		if err != nil {
			log.Println("Error converting text to bytes")
			continue
		}
		if len(textBytes) > bufferSize {
			log.Println("Message to long")
			continue
		}
		log.Printf("Sending message %d bytes long", len(textBytes))
		messageBytes := make([]byte, bufferSize)
		copy(messageBytes, textBytes)
		_, err = connection.Write(messageBytes)
		if err != nil {
			log.Println("Error sending message")
			continue
		}
	}
}

func listenForMessages(connection net.Conn) {
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