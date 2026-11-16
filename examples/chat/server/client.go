package main

import (
	"bufio"
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"training.pl/go/examples/chat/common"
)

// Client represents a connected client
type Client struct {
	ID       string
	Conn     net.Conn
	SendChan chan *common.Message
	Server   *Server

	// RemoteAddr is set once, before the pumps start, and only read afterwards.
	RemoteAddr string

	// nickname, status and rooms are read by the goroutines of OTHER clients
	// (BroadcastMessage, BroadcastUserList), so they are unexported and accessed
	// only through the Nickname/SetNickname, Status/SetStatus accessors.
	nickname string
	status   common.UserStatus
	rooms    map[string]bool
	mutex    sync.RWMutex

	// done instead of closing SendChan. A send channel is closed only by the
	// SENDER, and here there are many senders (every client broadcasting a
	// message). Closing it in Close() caused a "send on closed channel" panic that
	// took down the whole server - a select with default guards only against a
	// full channel, not a closed one.
	done      chan struct{}
	closeOnce sync.Once
}

// NewClient creates a new client instance
func NewClient(conn net.Conn, server *Server) *Client {
	return &Client{
		ID:       common.GenerateID("client"),
		Conn:     conn,
		status:   common.StatusActive,
		rooms:    make(map[string]bool),
		SendChan: make(chan *common.Message, 256),
		Server:   server,
		done:     make(chan struct{}),
	}
}

// Nickname returns the client's nickname (safe for concurrent use)
func (c *Client) Nickname() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.nickname
}

// SetNickname updates the client's nickname
func (c *Client) SetNickname(nickname string) {
	c.mutex.Lock()
	c.nickname = nickname
	c.mutex.Unlock()
}

// takeNickname clears the nickname and returns the previous value. It is the
// atomic "claim" that makes UnregisterClient idempotent: only the first caller
// gets a non-empty result.
func (c *Client) takeNickname() string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	nickname := c.nickname
	c.nickname = ""
	return nickname
}

// Status returns the client's current status
func (c *Client) Status() common.UserStatus {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.status
}

// SetStatus updates the client's status
func (c *Client) SetStatus(status common.UserStatus) {
	c.mutex.Lock()
	c.status = status
	c.mutex.Unlock()
}

// AddRoom adds a room to the client's room list
func (c *Client) AddRoom(roomID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.rooms[roomID] = true
}

// RemoveRoom removes a room from the client's room list
func (c *Client) RemoveRoom(roomID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.rooms, roomID)
}

// IsInRoom checks if the client is in a specific room
func (c *Client) IsInRoom(roomID string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.rooms[roomID]
}

// SendMessage sends a message to the client.
//
// The channel is never closed, so a send cannot panic; a finished client is
// recognised by its closed done channel.
func (c *Client) SendMessage(msg *common.Message) {
	select {
	case <-c.done:
		return
	default:
	}

	// No `case <-c.done` here: with a default branch present it would be
	// unreachable, because default is taken as soon as the send would block. The
	// policy is deliberately "drop when the buffer is full" - a slow client must
	// not stall whoever is broadcasting.
	select {
	case c.SendChan <- msg:
	default:
		log.Printf("Client %s send channel full, dropping message", c.Nickname())
	}
}

// ReadPump reads messages from the client connection
func (c *Client) ReadPump() {
	defer func() {
		c.Server.UnregisterClient(c)
		c.Close()
	}()

	scanner := bufio.NewScanner(c.Conn)
	// The initial 64 KiB is only the starting buffer; the maximum token size is
	// the constant shared with the client so both sides agree on the limit.
	scanner.Buffer(make([]byte, 0, 64*1024), common.MaxScannerBuffer)

	for scanner.Scan() {
		// Reset read deadline on successful read. A failure here means the
		// connection is already dead; the next Scan will report it.
		_ = c.Conn.SetReadDeadline(time.Now().Add(common.ReadTimeout))
		data := scanner.Bytes()
		msg, err := common.DecodeMessage(data)
		if err != nil {
			log.Printf("Error decoding message from %s: %v", c.Nickname(), err)
			continue
		}

		// Set sender to client's nickname
		msg.Sender = c.Nickname()
		msg.Timestamp = time.Now()

		// Handle the message
		if err := c.Server.HandleMessage(c, msg); err != nil {
			nickname := c.Nickname()
			log.Printf("Error handling message from %s: %v", nickname, err)
			// Send error message back to client
			errMsg := common.NewErrorMessage("Server", nickname, err.Error())
			c.SendMessage(errMsg)
		}
	}

	// net.ErrClosed means WE closed the socket (disconnect, rejection, shutdown)
	// and the pending read was interrupted - not worth an error line.
	if err := scanner.Err(); err != nil && !errors.Is(err, net.ErrClosed) {
		log.Printf("Error reading from %s: %v", c.Nickname(), err)
	}
}

// writeMessage encodes and writes one message to the connection.
func (c *Client) writeMessage(msg *common.Message) error {
	data, err := msg.Encode()
	if err != nil {
		// An encoding error is a bug in the message, not in the connection -
		// log it and carry on with the next message.
		log.Printf("Error encoding message: %v", err)
		return nil
	}
	if err := c.Conn.SetWriteDeadline(time.Now().Add(common.WriteTimeout)); err != nil {
		return err
	}
	_, err = c.Conn.Write(append(data, '\n'))
	return err
}

// WritePump writes messages to the client connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(common.KeepAliveInterval)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case <-c.done:
			// Flush whatever is already queued before closing, so that a final
			// error or goodbye message (e.g. the reason a registration was
			// rejected) still reaches the client. Non-blocking: nothing new is
			// awaited, only what is buffered right now is written.
			c.flushQueued()
			return

		case msg := <-c.SendChan:
			if err := c.writeMessage(msg); err != nil {
				log.Printf("Error writing to %s: %v", c.Nickname(), err)
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			ping := &common.Message{
				Type:      common.TypeAck,
				Timestamp: time.Now(),
			}
			if err := c.writeMessage(ping); err != nil {
				return
			}
		}
	}
}

// flushQueued writes the messages currently buffered in SendChan and returns as
// soon as the channel is empty or a write fails.
func (c *Client) flushQueued() {
	for {
		select {
		case msg := <-c.SendChan:
			if err := c.writeMessage(msg); err != nil {
				return
			}
		default:
			return
		}
	}
}

// Start runs both pumps and BLOCKS until they finish, so the caller
// (handleNewConnection) knows when the connection has really ended and can release
// its slot in the rate limiter at that point.
func (c *Client) Start() {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		c.WritePump()
	}()
	go func() {
		defer wg.Done()
		c.ReadPump()
	}()
	wg.Wait()
}

// Close signals completion. It is idempotent thanks to sync.Once - Close is
// called from ReadPump, from the DISCONNECT handler and from handleShutdown.
//
// The connection itself is closed by WritePump once it has flushed the queue
// (it observes the closed done channel); closing it here as well would cut off
// the messages still waiting to be written. ReadPump exits because the socket
// goes away, and if it were the one that called Close it has already returned.
// SendChan is deliberately not closed: that would be the receiver closing it, and
// the senders (other clients) would panic on their next send.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
	})
}
