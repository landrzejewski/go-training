package chat

import "net"

const bufferSize = 128

type Message struct {
	sender net.Conn
	bytes  []byte
}