package common

import "time"

// Connection limits
const (
	MaxConnections      = 100
	MaxConnectionsPerIP = 5
	ConnectionTimeout   = 30 * time.Second
	ReadTimeout         = 60 * time.Second
	WriteTimeout        = 60 * time.Second
	KeepAliveInterval   = 30 * time.Second
)

// Message limits
const (
	MaxMessageSize    = 4096
	MaxNicknameLength = 20
	MinNicknameLength = 3
	MaxRoomNameLength = 30
	MinRoomNameLength = 3
	MaxFileSize       = 100 * 1024 * 1024 // 100MB
	MaxFileNameLength = 255
	FileChunkSize     = 8192
	MaxScannerBuffer  = 1024 * 1024 // 1MB
)

// Rate limits
const (
	MessagesPerSecond    = 10
	RoomsPerUser         = 5
	FileTransfersPerUser = 3
)

// Timeouts
const (
	FileTransferTimeout = 5 * time.Minute
	EmptyRoomTimeout    = 30 * time.Minute
	ShutdownTimeout     = 30 * time.Second
)

// Validation patterns
const (
	NicknamePattern = "^[a-zA-Z0-9_-]+$"
	RoomNamePattern = "^[a-zA-Z0-9_\\- ]+$"
)
