package main

import (
	"sync"
	"time"

	"training.pl/go/examples/chat/common"
)

// Room represents a private chat room.
//
// ID, Name, Creator and CreatedAt are set once in NewRoom and never change, so
// they may be read without the lock. Everything else is guarded by mutex and is
// therefore unexported - go through the methods.
type Room struct {
	ID        string
	Name      string
	Creator   string
	CreatedAt time.Time

	description string
	members     map[string]bool
	invitations map[string]bool
	// lastEmptyAt is the moment the room lost its last member (the zero time while
	// the room is not empty). Empty-room cleanup is based on this timestamp, not on
	// CreatedAt - see CleanupManager.cleanupEmptyRooms and
	// RoomManager.EmptyRoomsOlderThan.
	lastEmptyAt time.Time
	mutex       sync.RWMutex
}

// NewRoom creates a new room
func NewRoom(name, creator string) *Room {
	return &Room{
		ID:          common.GenerateID("room"),
		Name:        name,
		Creator:     creator,
		CreatedAt:   time.Now(),
		members:     map[string]bool{creator: true},
		invitations: make(map[string]bool),
		// The creator is already in members above, so the room is NOT empty and
		// lastEmptyAt must stay zero. (It used to be set to time.Now() here, with a
		// comment claiming the creator was not a member yet - which contradicted the
		// line right above it.)
		lastEmptyAt: time.Time{},
	}
}

// AddMember adds a member to the room
func (r *Room) AddMember(nickname string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.members[nickname] = true
	r.lastEmptyAt = time.Time{} // the room is no longer empty
	delete(r.invitations, nickname)
}

// RemoveMember removes a member from the room
func (r *Room) RemoveMember(nickname string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.members, nickname)
	if len(r.members) == 0 {
		r.lastEmptyAt = time.Now()
	}
}

// IsMember checks if a user is a member of the room
func (r *Room) IsMember(nickname string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.members[nickname]
}

// InviteUser adds a user to the invitation list
func (r *Room) InviteUser(nickname string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.invitations[nickname] = true
}

// RevokeInvitation removes a pending invitation (e.g. when it is declined).
func (r *Room) RevokeInvitation(nickname string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.invitations, nickname)
}

// IsInvited checks if a user is invited to the room
func (r *Room) IsInvited(nickname string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.invitations[nickname]
}

// Members returns a list of room members
func (r *Room) Members() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	members := make([]string, 0, len(r.members))
	for member := range r.members {
		members = append(members, member)
	}
	return members
}

// SetDescription sets the room description
func (r *Room) SetDescription(description string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.description = description
}

// Description returns the room description
func (r *Room) Description() string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.description
}

// emptyFor reports whether the room has had no members for longer than d,
// measured at instant now.
func (r *Room) emptyFor(now time.Time, d time.Duration) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.members) == 0 && !r.lastEmptyAt.IsZero() && now.Sub(r.lastEmptyAt) > d
}

// RoomManager manages all rooms
type RoomManager struct {
	rooms map[string]*Room
	mutex sync.RWMutex
}

// NewRoomManager creates a new room manager
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
	}
}

// CreateRoom creates a new room
func (rm *RoomManager) CreateRoom(name, creator string) *Room {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	room := NewRoom(name, creator)
	rm.rooms[room.ID] = room
	return room
}

// Room retrieves a room by ID
func (rm *RoomManager) Room(roomID string) (*Room, bool) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	room, exists := rm.rooms[roomID]
	return room, exists
}

// UserRooms returns all rooms a user is member of
func (rm *RoomManager) UserRooms(nickname string) []*Room {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	var userRooms []*Room
	for _, room := range rm.rooms {
		if room.IsMember(nickname) {
			userRooms = append(userRooms, room)
		}
	}
	return userRooms
}

// EmptyRoomsOlderThan returns the rooms that have stood empty for longer than d.
// It exists so that CleanupManager does not have to reach into the manager's map
// and lock.
func (rm *RoomManager) EmptyRoomsOlderThan(d time.Duration) []*Room {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	now := time.Now()
	var stale []*Room
	for _, room := range rm.rooms {
		if room.emptyFor(now, d) {
			stale = append(stale, room)
		}
	}
	return stale
}

// RemoveRoom removes a room
func (rm *RoomManager) RemoveRoom(roomID string) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	delete(rm.rooms, roomID)
}

// BroadcastToRoom sends a message to all room members
func (rm *RoomManager) BroadcastToRoom(server *Server, roomID string, msg *common.Message) {
	room, exists := rm.Room(roomID)
	if !exists {
		return
	}

	members := room.Members()
	for _, member := range members {
		if client, ok := server.Client(member); ok {
			// Don't send to invisible users unless they're the sender
			if client.Status() == common.StatusInvisible && member != msg.Sender {
				continue
			}
			client.SendMessage(msg)
		}
	}
}
