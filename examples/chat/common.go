package chat

import "net"

const bufferSize = 128

type message struct {
	sender net.Conn
	bytes  []byte
}