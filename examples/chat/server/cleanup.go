package main

import (
	"log"
	"sync"
	"time"

	"training.pl/go/examples/chat/common"
)

// CleanupManager handles periodic cleanup of resources
type CleanupManager struct {
	server   *Server
	ticker   *time.Ticker
	stopChan chan struct{}
	stopOnce sync.Once
}

// NewCleanupManager creates a new cleanup manager
func NewCleanupManager(server *Server) *CleanupManager {
	return &CleanupManager{
		server:   server,
		ticker:   time.NewTicker(1 * time.Minute),
		stopChan: make(chan struct{}),
	}
}

// Start begins the cleanup routine
func (cm *CleanupManager) Start() {
	go cm.run()
}

// Stop stops the cleanup routine. It is safe to call more than once: closing an
// already closed channel panics, and RateLimiter.Stop in this package already
// guards against that with a sync.Once - this now matches it.
func (cm *CleanupManager) Stop() {
	cm.stopOnce.Do(func() {
		cm.ticker.Stop()
		close(cm.stopChan)
	})
}

// run executes periodic cleanup tasks
func (cm *CleanupManager) run() {
	for {
		select {
		case <-cm.ticker.C:
			cm.cleanupFileTransfers()
			cm.cleanupEmptyRooms()
		case <-cm.stopChan:
			return
		}
	}
}

// cleanupFileTransfers removes stale file transfers
func (cm *CleanupManager) cleanupFileTransfers() {
	now := time.Now()
	var toDelete []string

	// Find stale transfers
	cm.server.fileTransfers.Range(func(key, value any) bool {
		fileID := key.(string)
		ft := value.(*common.FileTransfer)

		// Check if transfer is older than timeout
		if now.Sub(ft.StartTime) > common.FileTransferTimeout {
			toDelete = append(toDelete, fileID)
			log.Printf("Cleaning up stale file transfer: %s", fileID)

			// Notify sender about timeout
			if sender, ok := cm.server.Client(ft.Sender); ok {
				errMsg := common.NewErrorMessage("Server", ft.Sender,
					"File transfer timed out: "+ft.Filename)
				sender.SendMessage(errMsg)
			}

			// Notify recipient about timeout
			if recipient, ok := cm.server.Client(ft.Recipient); ok {
				errMsg := common.NewErrorMessage("Server", ft.Recipient,
					"File transfer timed out: "+ft.Filename)
				recipient.SendMessage(errMsg)
			}

			// Clean up rate limiter
			cm.server.rateLimiter.RemoveFileTransfer(ft.Sender)
		}
		return true
	})

	// Delete stale transfers
	for _, fileID := range toDelete {
		cm.server.fileTransfers.Delete(fileID)
	}
}

// cleanupEmptyRooms removes rooms that have stood empty longer than EmptyRoomTimeout.
//
// What counts is the time SINCE THE ROOM BECAME EMPTY (Room.lastEmptyAt), not
// since it was created. The previous version compared CreatedAt, so a 31-minute-old
// room vanished the moment its last member left, while a one-minute-old room
// survived empty - even though the comment promised something entirely different.
func (cm *CleanupManager) cleanupEmptyRooms() {
	for _, room := range cm.server.roomManager.EmptyRoomsOlderThan(common.EmptyRoomTimeout) {
		log.Printf("Cleaning up empty room: %s (%s)", room.Name, room.ID)
		// Release the creator's room quota - see the comment on RoomDelete.
		cm.server.rateLimiter.RemoveRoom(room.Creator)
		cm.server.roomManager.RemoveRoom(room.ID)
	}
}
