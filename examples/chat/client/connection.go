package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"training.pl/go/examples/chat/common"
)

// ErrNotConnected is returned by the Send* methods when the connection has been
// shut down and the message can no longer be queued.
var ErrNotConnected = errors.New("not connected")

// Connection manages the server connection.
//
// Lock hierarchy: mutex guards the connection state (conn, connected, the
// per-dial cancel func); transfersMu guards only the fileTransfers map. The
// fields of an individual FileTransferProgress are guarded by that record's own
// mutex, never by either of the Connection locks.
type Connection struct {
	conn      net.Conn
	nickname  string
	status    common.UserStatus
	sendChan  chan *common.Message
	recvChan  chan *common.Message
	connected bool
	mutex     sync.RWMutex

	fileTransfers map[string]*FileTransferProgress
	transfersMu   sync.Mutex

	connectedChan chan struct{}

	// ctx lives as long as the Connection object and is cancelled by Disconnect.
	// Each dial derives its own child context (dialCancel) for the pumps, so a
	// reconnect can stop the old pumps without tearing down the whole object.
	ctx        context.Context
	cancel     context.CancelFunc
	dialCancel context.CancelFunc
}

// FileTransferProgress tracks file transfer progress. FileID, Filename,
// Filesize, IsIncoming and StartTime are set once at creation and immutable
// afterwards; progress, chunks and totalChunks are guarded by mutex.
type FileTransferProgress struct {
	FileID     string
	Filename   string
	Filesize   int64
	IsIncoming bool
	StartTime  time.Time

	mutex       sync.Mutex
	progress    float64
	chunks      map[int][]byte
	totalChunks int
}

// setProgress records the completion percentage.
func (t *FileTransferProgress) setProgress(p float64) {
	t.mutex.Lock()
	t.progress = p
	t.mutex.Unlock()
}

// Progress returns the completion percentage.
func (t *FileTransferProgress) Progress() float64 {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.progress
}

// addChunk stores an incoming chunk and returns the updated progress.
func (t *FileTransferProgress) addChunk(chunkNum, totalChunks int, data []byte) float64 {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	// A chunk may arrive before we have processed the init message.
	if t.totalChunks == 0 && totalChunks > 0 {
		t.totalChunks = totalChunks
	}
	t.chunks[chunkNum] = data
	// Guard against division by zero (it produced +Inf in the message).
	if t.totalChunks > 0 {
		t.progress = float64(len(t.chunks)) / float64(t.totalChunks) * 100
	}
	return t.progress
}

// orderedChunks returns the received chunks in order, or the index of the first
// missing chunk.
func (t *FileTransferProgress) orderedChunks() (chunks [][]byte, missing int) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	chunks = make([][]byte, 0, t.totalChunks)
	for i := 0; i < t.totalChunks; i++ {
		chunk, ok := t.chunks[i]
		if !ok {
			return nil, i
		}
		chunks = append(chunks, chunk)
	}
	return chunks, -1
}

// NewConnection creates a new connection manager
func NewConnection(nickname string) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		nickname:      nickname,
		status:        common.StatusActive,
		sendChan:      make(chan *common.Message, 100),
		recvChan:      make(chan *common.Message, 100),
		fileTransfers: make(map[string]*FileTransferProgress),
		connectedChan: make(chan struct{}, 1),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Nickname returns the nickname this connection registers with.
func (c *Connection) Nickname() string {
	return c.nickname
}

// Connect establishes connection to the server
func (c *Connection) Connect(address string) error {
	// Stop the pumps of any previous dial. dialCancel is read under c.mutex in
	// Disconnect(), so swapping it must happen under the lock too.
	c.mutex.Lock()
	if c.dialCancel != nil {
		c.dialCancel()
	}
	dialCtx, dialCancel := context.WithCancel(c.ctx)
	c.dialCancel = dialCancel
	c.mutex.Unlock()

	// Set connection timeout
	conn, err := net.DialTimeout("tcp", address, common.ConnectionTimeout)
	if err != nil {
		dialCancel()
		return err
	}

	c.mutex.Lock()
	c.conn = conn
	c.connected = true
	c.mutex.Unlock()

	// Set the initial read deadline; sendMessage sets the write deadline before
	// every write. A failure here means the socket is already unusable and the
	// first read will report it.
	_ = conn.SetReadDeadline(time.Now().Add(common.ReadTimeout))

	// Send connection message with nickname
	connectMsg := &common.Message{
		Type:    common.TypeConnect,
		Content: c.nickname,
	}

	if err := c.sendMessage(connectMsg); err != nil {
		conn.Close()
		dialCancel()
		return err
	}

	// Signal only AFTER the CONNECT message was sent successfully - previously
	// WaitForConnection() could also return when sendMessage then failed.
	select {
	case c.connectedChan <- struct{}{}:
	default:
	}

	// Start read and write pumps with context.
	// ctx and conn are passed as values read earlier under the lock - reading
	// c.ctx / c.conn here would race with another Connect().
	go c.readPump(dialCtx, conn)
	go c.writePump(dialCtx, conn)

	return nil
}

// ConnectWithBackoff dials the server until the FIRST successful connection,
// waiting with exponential backoff between attempts. It returns as soon as a
// connection is established or when Disconnect cancels the context.
//
// It does NOT reconnect if the connection drops later: once the pumps have
// started, a lost connection ends the session (readPump marks it disconnected).
func (c *Connection) ConnectWithBackoff(address string) {
	backoff := time.Second
	maxBackoff := time.Minute

	for {
		log.Printf("Connecting to %s...", address)
		err := c.Connect(address)
		if err == nil {
			log.Println("Connected successfully!")
			return
		}

		log.Printf("Connection failed: %v. Retrying in %v...", err, backoff)
		select {
		case <-time.After(backoff):
		case <-c.ctx.Done():
			return
		}

		// Exponential backoff
		backoff = min(2*backoff, maxBackoff)
	}
}

// IsConnected returns connection status
func (c *Connection) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected
}

// WaitForConnection blocks until connected. It returns ErrNotConnected if the
// connection is shut down (Disconnect) before that happens, so a caller is not
// left waiting forever.
func (c *Connection) WaitForConnection() error {
	select {
	case <-c.connectedChan:
		return nil
	case <-c.ctx.Done():
		return ErrNotConnected
	}
}

// SetConnected sets connection status
func (c *Connection) SetConnected(connected bool) {
	c.mutex.Lock()
	c.connected = connected
	c.mutex.Unlock()
}

// Disconnect closes the connection and stops any pending ConnectWithBackoff.
func (c *Connection) Disconnect() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Cancel the root context: this stops the pumps, the backoff loop and makes
	// every queued Send* return ErrNotConnected.
	c.cancel()

	if c.conn != nil {
		if err := c.conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			log.Printf("Error closing connection: %v", err)
		}
		c.connected = false
	}
}

// enqueue hands a message to writePump. It never blocks forever: when the
// connection is shut down (or was never established and Disconnect was called)
// the context is cancelled and ErrNotConnected is returned instead - a bare
// `c.sendChan <- msg` used to hang the UI goroutine once writePump had exited.
func (c *Connection) enqueue(msg *common.Message) error {
	select {
	case c.sendChan <- msg:
		return nil
	case <-c.ctx.Done():
		return ErrNotConnected
	}
}

// SendTextMessage sends a text message
func (c *Connection) SendTextMessage(recipient, content string) error {
	return c.enqueue(common.NewTextMessage(c.nickname, recipient, content))
}

// SendBroadcastMessage sends a broadcast message
func (c *Connection) SendBroadcastMessage(content string) error {
	return c.enqueue(common.NewBroadcastMessage(c.nickname, content))
}

// SendRoomMessage sends a message to a room
func (c *Connection) SendRoomMessage(roomID, content string) error {
	return c.enqueue(&common.Message{
		Type:      common.TypeText,
		Room:      roomID,
		Content:   content,
		Timestamp: time.Now(),
	})
}

// SendDisconnect tells the server we are leaving on purpose.
func (c *Connection) SendDisconnect(reason string) error {
	return c.enqueue(&common.Message{
		Type:      common.TypeDisconnect,
		Sender:    c.nickname,
		Content:   reason,
		Timestamp: time.Now(),
	})
}

// ChangeStatus updates user status
func (c *Connection) ChangeStatus(status common.UserStatus) error {
	c.mutex.Lock()
	c.status = status
	c.mutex.Unlock()
	return c.enqueue(common.NewStatusMessage(c.nickname, status))
}

// CreateRoom creates a new room
func (c *Connection) CreateRoom(name string) error {
	return c.enqueue(&common.Message{
		Type:      common.TypeRoom,
		Action:    common.RoomCreate,
		Content:   name,
		Timestamp: time.Now(),
	})
}

// InviteToRoom invites a user to a room
func (c *Connection) InviteToRoom(roomID, userNickname string) error {
	return c.enqueue(&common.Message{
		Type:      common.TypeInvite,
		Room:      roomID,
		Recipient: userNickname,
		Timestamp: time.Now(),
	})
}

// RespondToInvite responds to a room invitation
func (c *Connection) RespondToInvite(roomID string, accept bool) error {
	response := "decline"
	if accept {
		response = "accept"
	}

	return c.enqueue(&common.Message{
		Type:      common.TypeInviteResp,
		Room:      roomID,
		Content:   response,
		Timestamp: time.Now(),
	})
}

// LeaveRoom sends a leave room message
func (c *Connection) LeaveRoom(roomID string) error {
	return c.enqueue(&common.Message{
		Type:      common.TypeRoom,
		Action:    common.RoomLeave,
		Room:      roomID,
		Timestamp: time.Now(),
	})
}

// RoomMembers requests the member list for a room
func (c *Connection) RoomMembers(roomID string) error {
	return c.enqueue(&common.Message{
		Type:      common.TypeRoom,
		Action:    common.RoomMembers,
		Room:      roomID,
		Timestamp: time.Now(),
	})
}

// KickFromRoom kicks a user from a room (creator only)
func (c *Connection) KickFromRoom(roomID, nickname string) error {
	return c.enqueue(&common.Message{
		Type:      common.TypeRoom,
		Action:    common.RoomKick,
		Room:      roomID,
		Recipient: nickname,
		Timestamp: time.Now(),
	})
}

// DeleteRoom deletes a room (creator only)
func (c *Connection) DeleteRoom(roomID string) error {
	return c.enqueue(&common.Message{
		Type:      common.TypeRoom,
		Action:    common.RoomDelete,
		Room:      roomID,
		Timestamp: time.Now(),
	})
}

// SetRoomTopic sets the topic/description for a room
func (c *Connection) SetRoomTopic(roomID, description string) error {
	return c.enqueue(&common.Message{
		Type:      common.TypeRoom,
		Action:    common.RoomSetTopic,
		Room:      roomID,
		Content:   description,
		Timestamp: time.Now(),
	})
}

// Messages returns the receive channel
func (c *Connection) Messages() <-chan *common.Message {
	return c.recvChan
}

// readPump reads messages from the server.
//
// It takes the net.Conn as a parameter instead of reading the c.conn field,
// which Connect() writes under c.mutex - otherwise it would be a data race.
//
// Cancellation: scanner.Scan blocks inside a socket read and cannot observe ctx
// by itself. The ctx check in the loop only runs between messages; what really
// ends the goroutine on Disconnect is conn.Close() (called by Disconnect and by
// the writePump defer), which makes the pending Scan return an error.
func (c *Connection) readPump(ctx context.Context, conn net.Conn) {
	defer func() {
		c.SetConnected(false)
		conn.Close()
	}()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), common.MaxScannerBuffer)

	for scanner.Scan() {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Reset read deadline on successful read. If this fails the socket is
		// already dead and the next Scan reports it.
		_ = conn.SetReadDeadline(time.Now().Add(common.ReadTimeout))

		data := scanner.Bytes()
		msg, err := common.DecodeMessage(data)
		if err != nil {
			log.Printf("Error decoding message: %v", err)
			continue
		}

		// Handle file chunks separately
		if msg.Type == common.TypeFileChunk {
			c.handleFileChunk(ctx, msg)
			continue
		}

		// The init message carries the file name, size and chunk count - the
		// chunks themselves do not. Without registering the transfer here the
		// record was created only from the first chunk, with an EMPTY name, so
		// writing it failed with an error "invalid filename".
		//
		// Our OWN init comes back too (the server echoes it with the assigned
		// FileID) - that one is an outgoing transfer and is handled by the UI via
		// FileTransfer.startOutgoing.
		if msg.Type == common.TypeFile && msg.Sender != c.nickname {
			c.registerIncomingTransfer(msg)
		}
		// select with ctx.Done(): if the UI stopped receiving, a bare
		// `c.recvChan <- msg` would block this goroutine forever.
		select {
		case c.recvChan <- msg:
		case <-ctx.Done():
			return
		}
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, net.ErrClosed) {
		log.Printf("Read error: %v", err)
	}
}

// writePump writes messages to the server.
//
// Like readPump it receives conn as a parameter rather than reading the c.conn
// field: Connect() writes that field under c.mutex, so touching it here would be a
// data race - and after a reconnect the deferred Close would shut down the NEW
// connection.
func (c *Connection) writePump(ctx context.Context, conn net.Conn) {
	ticker := time.NewTicker(common.KeepAliveInterval)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			// Flush what is already queued (e.g. the DISCONNECT sent on quit)
			// before the socket goes away. Non-blocking: only what is buffered
			// right now is written.
			c.flushQueued()
			return
		case msg := <-c.sendChan:
			if err := c.sendMessage(msg); err != nil {
				log.Printf("Write error: %v", err)
				return
			}

		case <-ticker.C:
			// A real keep-alive: the SERVER resets its read deadline only after a
			// successful read, so something has to be SENT to it. Merely setting a
			// write deadline (the previous version) sent nothing, so a silent user
			// was disconnected after ReadTimeout (60 s).
			keepAlive := &common.Message{
				Type:      common.TypeAck,
				Sender:    c.nickname,
				Timestamp: time.Now(),
			}
			if err := c.sendMessage(keepAlive); err != nil {
				log.Printf("Keep-alive error: %v", err)
				return
			}
		}
	}
}

// flushQueued writes the messages currently buffered in sendChan and returns as
// soon as the channel is empty or a write fails.
func (c *Connection) flushQueued() {
	for {
		select {
		case msg := <-c.sendChan:
			if err := c.sendMessage(msg); err != nil {
				return
			}
		default:
			return
		}
	}
}

// sendMessage sends a message to the server
func (c *Connection) sendMessage(msg *common.Message) error {
	data, err := msg.Encode()
	if err != nil {
		return err
	}

	c.mutex.RLock()
	conn := c.conn
	c.mutex.RUnlock()

	if conn == nil {
		return ErrNotConnected
	}

	if err := conn.SetWriteDeadline(time.Now().Add(common.WriteTimeout)); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}

	_, err = conn.Write(append(data, '\n'))
	return err
}

// registerIncomingTransfer records the metadata of an incoming transfer before its
// chunks arrive.
func (c *Connection) registerIncomingTransfer(msg *common.Message) {
	c.transfersMu.Lock()
	defer c.transfersMu.Unlock()
	if _, exists := c.fileTransfers[msg.FileID]; exists {
		return
	}
	c.fileTransfers[msg.FileID] = newIncomingTransfer(msg)
}

// newIncomingTransfer builds the record for a transfer we are receiving.
func newIncomingTransfer(msg *common.Message) *FileTransferProgress {
	return &FileTransferProgress{
		FileID:      msg.FileID,
		Filename:    msg.Filename,
		Filesize:    msg.Filesize,
		IsIncoming:  true,
		StartTime:   time.Now(),
		chunks:      make(map[int][]byte),
		totalChunks: msg.TotalChunks,
	}
}

// transfer looks up a transfer record by ID.
func (c *Connection) transfer(fileID string) (*FileTransferProgress, bool) {
	c.transfersMu.Lock()
	defer c.transfersMu.Unlock()
	t, ok := c.fileTransfers[fileID]
	return t, ok
}

// removeTransfer deletes a transfer record and returns it.
func (c *Connection) removeTransfer(fileID string) (*FileTransferProgress, bool) {
	c.transfersMu.Lock()
	defer c.transfersMu.Unlock()
	t, ok := c.fileTransfers[fileID]
	delete(c.fileTransfers, fileID)
	return t, ok
}

// handleFileChunk processes incoming file chunks
func (c *Connection) handleFileChunk(ctx context.Context, msg *common.Message) {
	c.transfersMu.Lock()
	transfer, exists := c.fileTransfers[msg.FileID]
	if !exists {
		// Chunk arrived before the init message - register from what we have.
		transfer = newIncomingTransfer(msg)
		c.fileTransfers[msg.FileID] = transfer
	}
	c.transfersMu.Unlock()

	// Store the chunk under the record's own lock; the map lock is released.
	progress := transfer.addChunk(msg.ChunkNum, msg.TotalChunks, msg.Data)

	// Forward to UI for progress display
	progressMsg := &common.Message{
		Type:     common.TypeFileChunk,
		FileID:   msg.FileID,
		Filename: transfer.Filename,
		Content:  fmt.Sprintf("%.1f%%", progress),
	}
	select {
	case c.recvChan <- progressMsg:
	case <-ctx.Done():
		return
	}

	// Completion is signalled by the SERVER (TypeFileComplete). The client used to
	// add its own message as well, so a file was "received" twice - the second time
	// the transfer record no longer existed and the user saw a confusing
	// "Error saving file: file transfer not found".
}
