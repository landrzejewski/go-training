package common

import (
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// MessageType represents the type of message being sent
type MessageType string

const (
	// Message types
	TypeText         MessageType = "TEXT"
	TypeFile         MessageType = "FILE"
	TypeFileChunk    MessageType = "FILE_CHUNK"
	TypeFileComplete MessageType = "FILE_COMPLETE"
	TypeStatus       MessageType = "STATUS"
	TypeRoom         MessageType = "ROOM"
	TypeInvite       MessageType = "INVITE"
	TypeInviteResp   MessageType = "INVITE_RESP"
	TypeUserList     MessageType = "USER_LIST"
	TypeError        MessageType = "ERROR"
	TypeConnect      MessageType = "CONNECT"
	TypeDisconnect   MessageType = "DISCONNECT"
	TypeAck          MessageType = "ACK"
)

// UserStatus represents the status of a user
type UserStatus string

const (
	StatusActive    UserStatus = "ACTIVE"
	StatusBusy      UserStatus = "BUSY"
	StatusInvisible UserStatus = "INVISIBLE"
)

// RoomAction represents actions related to rooms
type RoomAction string

const (
	RoomCreate       RoomAction = "CREATE"
	RoomJoin         RoomAction = "JOIN"
	RoomLeave        RoomAction = "LEAVE"
	RoomLeaveConfirm RoomAction = "LEAVE_CONFIRM"
	RoomMsg          RoomAction = "MSG"
	RoomMembers      RoomAction = "MEMBERS"
	RoomKick         RoomAction = "KICK"
	RoomDelete       RoomAction = "DELETE"
	RoomSetTopic     RoomAction = "TOPIC"
)

// Message represents a message in the chat protocol
type Message struct {
	Type      MessageType `json:"type"`
	Sender    string      `json:"sender"`
	Recipient string      `json:"recipient,omitempty"` // Empty for broadcast, "*" for all
	Room      string      `json:"room,omitempty"`
	// RoomName carries JUST the room name, separate from Content, where the server
	// puts the full sentence shown to the user. Without that split the client stored
	// the message text where it meant to store the name.
	RoomName string     `json:"roomName,omitempty"`
	Content  string     `json:"content,omitempty"`
	Status   UserStatus `json:"status,omitempty"`
	Action   RoomAction `json:"action,omitempty"`
	Filename string     `json:"filename,omitempty"`
	Filesize int64      `json:"filesize,omitempty"`
	FileID   string     `json:"file_id,omitempty"`
	// RefID is a client-side reference for an outgoing file transfer. The client
	// sends it in the FILE init message; the server, which mints the real FileID,
	// echoes the init back to the sender with both fields set so the sender can
	// pair the server-assigned ID with its local record. An ERROR rejecting the
	// init carries the same RefID.
	RefID       string    `json:"ref_id,omitempty"`
	ChunkNum    int       `json:"chunk_num,omitempty"`
	TotalChunks int       `json:"total_chunks,omitempty"`
	Data        []byte    `json:"data,omitempty"`
	Users       []string  `json:"users,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Error       string    `json:"error,omitempty"`
}

// NewTextMessage creates a new text message
func NewTextMessage(sender, recipient, content string) *Message {
	return &Message{
		Type:      TypeText,
		Sender:    sender,
		Recipient: recipient,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// NewBroadcastMessage creates a new broadcast message
func NewBroadcastMessage(sender, content string) *Message {
	return &Message{
		Type:      TypeText,
		Sender:    sender,
		Recipient: "*",
		Content:   content,
		Timestamp: time.Now(),
	}
}

// NewStatusMessage creates a new status update message
func NewStatusMessage(sender string, status UserStatus) *Message {
	return &Message{
		Type:      TypeStatus,
		Sender:    sender,
		Status:    status,
		Timestamp: time.Now(),
	}
}

// NewErrorMessage creates a new error message carrying reason as its text.
func NewErrorMessage(sender, recipient, reason string) *Message {
	return &Message{
		Type:      TypeError,
		Sender:    sender,
		Recipient: recipient,
		Error:     reason,
		Timestamp: time.Now(),
	}
}

// Encode serializes the message to JSON
func (m *Message) Encode() ([]byte, error) {
	return json.Marshal(m)
}

// DecodeMessage deserializes a JSON message. On error it returns a nil message,
// following the usual Go convention, so callers cannot accidentally use a
// half-filled value.
func DecodeMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ErrInvalidChunkCount is returned by NewFileTransfer for a non-positive chunk count.
var ErrInvalidChunkCount = errors.New("total chunk count must be positive")

// FileTransfer represents an ongoing file transfer. The exported fields are set
// once by NewFileTransfer and are read-only afterwards; the chunk map is private
// and only ever touched under the mutex.
type FileTransfer struct {
	FileID      string
	Filename    string
	Filesize    int64
	Sender      string
	Recipient   string
	TotalChunks int
	StartTime   time.Time

	receivedChunks map[int][]byte
	mutex          sync.RWMutex
}

// NewFileTransfer creates a transfer record. TotalChunks must be positive:
// a transfer with zero chunks would report IsComplete() immediately and a
// negative one could never complete.
func NewFileTransfer(fileID, filename string, filesize int64, sender, recipient string, totalChunks int, startTime time.Time) (*FileTransfer, error) {
	if totalChunks <= 0 {
		return nil, ErrInvalidChunkCount
	}
	return &FileTransfer{
		FileID:         fileID,
		Filename:       filename,
		Filesize:       filesize,
		Sender:         sender,
		Recipient:      recipient,
		TotalChunks:    totalChunks,
		StartTime:      startTime,
		receivedChunks: make(map[int][]byte),
	}, nil
}

// IsComplete checks if all chunks have been received
func (ft *FileTransfer) IsComplete() bool {
	ft.mutex.RLock()
	defer ft.mutex.RUnlock()
	return len(ft.receivedChunks) == ft.TotalChunks
}

// Progress returns the progress percentage
func (ft *FileTransfer) Progress() float64 {
	ft.mutex.RLock()
	defer ft.mutex.RUnlock()
	if ft.TotalChunks == 0 {
		return 0
	}
	return float64(len(ft.receivedChunks)) / float64(ft.TotalChunks) * 100
}

// ReceivedCount returns the number of chunks recorded so far.
func (ft *FileTransfer) ReceivedCount() int {
	ft.mutex.RLock()
	defer ft.mutex.RUnlock()
	return len(ft.receivedChunks)
}

// AddChunk adds a chunk to the file transfer (used on the RECEIVING side, which
// actually reassembles the file from its fragments).
func (ft *FileTransfer) AddChunk(chunkNum int, data []byte) {
	ft.mutex.Lock()
	defer ft.mutex.Unlock()
	ft.receivedChunks[chunkNum] = data
}

// MarkChunkReceived records only the FACT that a chunk arrived, without keeping
// its contents. The server merely relays the data, so holding the bytes would mean
// buffering whole files in memory for no reason.
func (ft *FileTransfer) MarkChunkReceived(chunkNum int) {
	ft.mutex.Lock()
	defer ft.mutex.Unlock()
	ft.receivedChunks[chunkNum] = nil
}

// Chunk retrieves a specific chunk
func (ft *FileTransfer) Chunk(chunkNum int) ([]byte, bool) {
	ft.mutex.RLock()
	defer ft.mutex.RUnlock()
	data, exists := ft.receivedChunks[chunkNum]
	return data, exists
}
