package main

import (
	"fmt"
	"net"
	"sync"
	"time"

	"training.pl/go/examples/chat/common"
)

// RateLimiter manages rate limiting for the server
type RateLimiter struct {
	// Connection limits
	totalConnections int
	connectionsByIP  map[string]int
	connMutex        sync.RWMutex

	// Message rate limiting
	messageRates map[string]*userRateLimit
	rateMutex    sync.RWMutex

	// Room creation limiting
	roomsPerUser map[string]int
	roomMutex    sync.RWMutex

	// File transfer limiting
	transfersPerUser map[string]int
	transferMutex    sync.RWMutex

	// Cleanup ticker
	cleanupTicker *time.Ticker
	stopChan      chan struct{}
	stopOnce      sync.Once
}

type userRateLimit struct {
	messages  int
	lastReset time.Time
	mutex     sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		connectionsByIP:  make(map[string]int),
		messageRates:     make(map[string]*userRateLimit),
		roomsPerUser:     make(map[string]int),
		transfersPerUser: make(map[string]int),
		cleanupTicker:    time.NewTicker(1 * time.Minute),
		stopChan:         make(chan struct{}),
	}

	// Start cleanup routine
	go rl.cleanup()

	return rl
}

// TryAddConnection checks the limits AND registers the connection in ONE operation
// under the same lock.
//
// Splitting that into CanConnect (in the accept loop) and AddConnection (in a
// separate goroutine) produced a classic TOCTOU: N concurrent connections all saw
// the state from before the increment and all passed the limit check.
func (rl *RateLimiter) TryAddConnection(addr net.Addr) error {
	rl.connMutex.Lock()
	defer rl.connMutex.Unlock()

	ip, err := ipOf(addr)
	if err != nil {
		return err
	}

	if err := rl.canConnectLocked(ip); err != nil {
		return err
	}

	rl.totalConnections++
	rl.connectionsByIP[ip]++
	return nil
}

// ipOf extracts the host part of a network address.
func ipOf(addr net.Addr) (string, error) {
	ip, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "", fmt.Errorf("invalid address format %q: %w", addr.String(), err)
	}
	return ip, nil
}

// canConnectLocked requires connMutex to be held.
func (rl *RateLimiter) canConnectLocked(ip string) error {
	// Check total connections
	if rl.totalConnections >= common.MaxConnections {
		return fmt.Errorf("server has reached maximum connection limit (%d)", common.MaxConnections)
	}

	// Check per-IP limit
	if rl.connectionsByIP[ip] >= common.MaxConnectionsPerIP {
		return fmt.Errorf("IP %s has reached maximum connection limit (%d)", ip, common.MaxConnectionsPerIP)
	}

	return nil
}

// RemoveConnection removes a connection. An address that cannot be parsed was
// never counted by TryAddConnection, so there is nothing to release for it.
func (rl *RateLimiter) RemoveConnection(addr net.Addr) {
	ip, err := ipOf(addr)
	if err != nil {
		return
	}

	rl.connMutex.Lock()
	defer rl.connMutex.Unlock()

	if rl.totalConnections > 0 {
		rl.totalConnections--
	}

	if count := rl.connectionsByIP[ip]; count > 0 {
		if count == 1 {
			delete(rl.connectionsByIP, ip)
		} else {
			rl.connectionsByIP[ip]--
		}
	}
}

// CanSendMessage checks if a user can send a message.
//
// NOTE: this is a FIXED WINDOW, not a token bucket. The counter resets a full
// second after the last reset, so across a window boundary up to
// 2*MessagesPerSecond messages can be sent in a very short span (10 at t=0.99 s and
// 10 at t=1.01 s). For training material that is an acceptable simplification -
// smoothing it out would need a real token bucket.
func (rl *RateLimiter) CanSendMessage(nickname string) error {
	rl.rateMutex.Lock()
	userLimit, exists := rl.messageRates[nickname]
	if !exists {
		userLimit = &userRateLimit{
			lastReset: time.Now(),
		}
		rl.messageRates[nickname] = userLimit
	}
	rl.rateMutex.Unlock()

	userLimit.mutex.Lock()
	defer userLimit.mutex.Unlock()

	// Reset counter if a second has passed
	if time.Since(userLimit.lastReset) >= time.Second {
		userLimit.messages = 0
		userLimit.lastReset = time.Now()
	}

	// Check rate limit
	if userLimit.messages >= common.MessagesPerSecond {
		return fmt.Errorf("message rate limit exceeded (%d messages per 1s window)", common.MessagesPerSecond)
	}

	userLimit.messages++
	return nil
}

// TryAddRoom checks the per-user room quota AND claims a slot in one operation
// under a single lock - the same TOCTOU fix as TryAddConnection. The old
// CanCreateRoom + AddRoom pair let two concurrent CREATE requests both pass the
// check before either incremented the counter.
func (rl *RateLimiter) TryAddRoom(nickname string) error {
	rl.roomMutex.Lock()
	defer rl.roomMutex.Unlock()

	if rl.roomsPerUser[nickname] >= common.RoomsPerUser {
		return fmt.Errorf("room creation limit exceeded (%d rooms per user)", common.RoomsPerUser)
	}

	rl.roomsPerUser[nickname]++
	return nil
}

// RemoveRoom removes a room from user's count
func (rl *RateLimiter) RemoveRoom(nickname string) {
	rl.roomMutex.Lock()
	defer rl.roomMutex.Unlock()

	if count := rl.roomsPerUser[nickname]; count > 0 {
		if count == 1 {
			delete(rl.roomsPerUser, nickname)
		} else {
			rl.roomsPerUser[nickname]--
		}
	}
}

// TryAddFileTransfer checks the per-user transfer quota AND claims a slot in one
// operation under a single lock (see TryAddRoom).
func (rl *RateLimiter) TryAddFileTransfer(nickname string) error {
	rl.transferMutex.Lock()
	defer rl.transferMutex.Unlock()

	if rl.transfersPerUser[nickname] >= common.FileTransfersPerUser {
		return fmt.Errorf("file transfer limit exceeded (%d concurrent transfers per user)", common.FileTransfersPerUser)
	}

	rl.transfersPerUser[nickname]++
	return nil
}

// RemoveFileTransfer removes a file transfer
func (rl *RateLimiter) RemoveFileTransfer(nickname string) {
	rl.transferMutex.Lock()
	defer rl.transferMutex.Unlock()

	if count := rl.transfersPerUser[nickname]; count > 0 {
		if count == 1 {
			delete(rl.transfersPerUser, nickname)
		} else {
			rl.transfersPerUser[nickname]--
		}
	}
}

// RemoveUser cleans up the CONNECTION-related rate limit state of a user who
// disconnected: the message window.
//
// The room and file-transfer counters are deliberately left alone. The rooms a
// user created still exist after a disconnect (they are only removed by the
// creator or by the empty-room cleanup), and each removal path releases its own
// slot via RemoveRoom / RemoveFileTransfer. Clearing the counters here let a user
// bypass the quota by simply reconnecting.
func (rl *RateLimiter) RemoveUser(nickname string) {
	rl.rateMutex.Lock()
	delete(rl.messageRates, nickname)
	rl.rateMutex.Unlock()
}

// cleanup periodically cleans up old rate limit data
// cleanup ends when stopChan is closed. cleanupTicker.Stop() alone does NOT close
// the C channel, so `for range rl.cleanupTicker.C` would block forever and the
// goroutine would leak after every Stop().
func (rl *RateLimiter) cleanup() {
	for {
		select {
		case <-rl.stopChan:
			return
		case <-rl.cleanupTicker.C:
			rl.rateMutex.Lock()
			for nick, userLimit := range rl.messageRates {
				userLimit.mutex.Lock()
				if time.Since(userLimit.lastReset) > 5*time.Minute {
					delete(rl.messageRates, nick)
				}
				userLimit.mutex.Unlock()
			}
			rl.rateMutex.Unlock()
		}
	}
}

// Stop stops the rate limiter. Idempotent - closing an already closed channel
// would panic.
func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() {
		rl.cleanupTicker.Stop()
		close(rl.stopChan)
	})
}
