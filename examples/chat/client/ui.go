package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"training.pl/go/examples/chat/common"
)

// UI handles terminal user interface
type UI struct {
	conn         *Connection
	fileTransfer *FileTransfer
	// quit is the shared shutdown routine (set by main): /quit and Ctrl-C must
	// do exactly the same thing.
	quit  func()
	rooms map[string]string // roomID -> roomName
	users []string
	mutex sync.RWMutex
}

// NewUI creates a new UI instance
func NewUI(conn *Connection, ft *FileTransfer, quit func()) *UI {
	return &UI{
		conn:         conn,
		fileTransfer: ft,
		quit:         quit,
		rooms:        make(map[string]string),
	}
}

// command describes one slash command for the help text.
type command struct {
	usage string
	help  string
}

// commands is the single table behind /help. Every entry must have a matching
// case in handleCommand / handleRoomCommand, so the help text cannot drift from
// what the client actually implements.
var commands = []command{
	{"/help", "Show help"},
	{"/users", "List online users"},
	{"/msg <nick> <message>", "Send private message"},
	{"/file <nick> <filepath>", "Send file"},
	{"/status <active|busy|invisible>", "Change status"},
	{"/room create <name>", "Create private room"},
	{"/room invite <id> <nick>", "Invite to room"},
	{"/room accept <id>", "Accept room invitation"},
	{"/room decline <id>", "Decline room invitation"},
	{"/room msg <id> <message>", "Message to room"},
	{"/room list", "List your rooms"},
	{"/room leave <id>", "Leave a room"},
	{"/room members <id>", "List room members"},
	{"/room kick <id> <nick>", "Kick a member (creator only)"},
	{"/room delete <id>", "Delete a room (creator only)"},
	{"/room topic <id> <text>", "Set the room topic"},
	{"/transfers", "Show file transfers"},
	{"/quit", "Exit"},
}

// Start starts the UI. It returns when standard input is closed.
func (ui *UI) Start() {
	// Clear screen and show welcome
	clearScreen()
	ui.showWelcome()

	// Start message receiver
	go ui.receiveMessages()

	// Start input handler
	ui.handleInput()
}

// Use platform-specific clearScreen function defined in platform_*.go files

// showWelcome displays welcome message
func (ui *UI) showWelcome() {
	fmt.Println("=================================")
	fmt.Println("   TCP Chat Client")
	fmt.Println("=================================")
	fmt.Printf("Connected as: %s\n", ui.conn.Nickname())
	fmt.Println("\nCommands:")
	for _, c := range commands {
		fmt.Printf("  %-33s - %s\n", c.usage, c.help)
	}
	fmt.Println("\nType messages without '/' to broadcast to all users")
	fmt.Printf("=================================\n\n")
}

// handleInput handles user input
func (ui *UI) handleInput() {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if strings.HasPrefix(input, "/") {
			ui.handleCommand(input)
		} else {
			// Send broadcast message
			ui.report(ui.conn.SendBroadcastMessage(input))
		}
	}
}

// report prints a send error, if any. Every Send* call can fail with
// ErrNotConnected once the connection is gone; silently ignoring that left the
// user typing into the void.
func (ui *UI) report(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

// handleCommand handles slash commands
func (ui *UI) handleCommand(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "/help":
		ui.showWelcome()

	case "/users":
		ui.showUsers()

	case "/msg":
		if len(parts) < 3 {
			fmt.Println("Usage: /msg <nickname> <message>")
			return
		}
		recipient := parts[1]
		message := strings.Join(parts[2:], " ")
		ui.report(ui.conn.SendTextMessage(recipient, message))

	case "/file":
		if len(parts) < 3 {
			fmt.Println("Usage: /file <nickname> <filepath>")
			return
		}
		recipient := parts[1]
		filepath := strings.Join(parts[2:], " ")

		if err := ui.fileTransfer.SendFile(recipient, filepath); err != nil {
			fmt.Printf("Error sending file: %v\n", err)
		} else {
			fmt.Printf("Sending file to %s...\n", recipient)
		}

	case "/status":
		if len(parts) < 2 {
			fmt.Println("Usage: /status <active|busy|invisible>")
			return
		}

		var status common.UserStatus
		switch strings.ToLower(parts[1]) {
		case "active":
			status = common.StatusActive
		case "busy":
			status = common.StatusBusy
		case "invisible":
			status = common.StatusInvisible
		default:
			fmt.Println("Invalid status. Use: active, busy, or invisible")
			return
		}

		if err := ui.conn.ChangeStatus(status); err != nil {
			ui.report(err)
			return
		}
		fmt.Printf("Status changed to: %s\n", status)

	case "/room":
		ui.handleRoomCommand(parts[1:])

	case "/transfers":
		ui.showTransfers()

	case "/quit":
		ui.quit()

	default:
		fmt.Printf("Unknown command: %s\n", command)
	}
}

// handleRoomCommand handles room-related commands
func (ui *UI) handleRoomCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: /room <create|invite|accept|decline|msg|list|leave|members|kick|delete|topic> ...")
		return
	}

	subcommand := strings.ToLower(args[0])

	switch subcommand {
	case "create":
		if len(args) < 2 {
			fmt.Println("Usage: /room create <name>")
			return
		}
		roomName := strings.Join(args[1:], " ")
		ui.report(ui.conn.CreateRoom(roomName))

	case "invite":
		if len(args) < 3 {
			fmt.Println("Usage: /room invite <room_id> <nickname>")
			return
		}
		roomID := args[1]
		nickname := args[2]
		ui.report(ui.conn.InviteToRoom(roomID, nickname))

	case "accept":
		if len(args) < 2 {
			fmt.Println("Usage: /room accept <room_id>")
			return
		}
		roomID := args[1]
		if err := ui.conn.RespondToInvite(roomID, true); err != nil {
			ui.report(err)
			return
		}
		fmt.Printf("Accepted invitation to room %s\n", roomID)

	case "decline":
		if len(args) < 2 {
			fmt.Println("Usage: /room decline <room_id>")
			return
		}
		roomID := args[1]
		if err := ui.conn.RespondToInvite(roomID, false); err != nil {
			ui.report(err)
			return
		}
		fmt.Printf("Declined invitation to room %s\n", roomID)

	case "msg":
		if len(args) < 3 {
			fmt.Println("Usage: /room msg <room_id> <message>")
			return
		}
		roomID := args[1]
		message := strings.Join(args[2:], " ")
		ui.report(ui.conn.SendRoomMessage(roomID, message))

	case "list":
		ui.showRooms()

	case "leave":
		if len(args) < 2 {
			fmt.Println("Usage: /room leave <room_id>")
			return
		}
		roomID := args[1]
		// Don't delete locally here - wait for server confirmation
		ui.report(ui.conn.LeaveRoom(roomID))

	case "members":
		if len(args) < 2 {
			fmt.Println("Usage: /room members <room_id>")
			return
		}
		roomID := args[1]
		ui.report(ui.conn.RoomMembers(roomID))

	case "kick":
		if len(args) < 3 {
			fmt.Println("Usage: /room kick <room_id> <nickname>")
			return
		}
		roomID := args[1]
		nickname := args[2]
		ui.report(ui.conn.KickFromRoom(roomID, nickname))

	case "delete":
		if len(args) < 2 {
			fmt.Println("Usage: /room delete <room_id>")
			return
		}
		roomID := args[1]
		ui.report(ui.conn.DeleteRoom(roomID))

	case "topic":
		if len(args) < 3 {
			fmt.Println("Usage: /room topic <room_id> <description>")
			return
		}
		roomID := args[1]
		description := strings.Join(args[2:], " ")
		ui.report(ui.conn.SetRoomTopic(roomID, description))

	default:
		fmt.Printf("Unknown room command: %s\n", subcommand)
	}
}

// receiveMessages handles incoming messages
func (ui *UI) receiveMessages() {
	for msg := range ui.conn.Messages() {
		ui.handleMessage(msg)
	}
}

// handleMessage processes incoming messages
func (ui *UI) handleMessage(msg *common.Message) {
	timestamp := msg.Timestamp.Format("15:04:05")
	me := ui.conn.Nickname()

	switch msg.Type {
	case common.TypeText:
		if msg.Room != "" {
			// Room message
			ui.mutex.RLock()
			roomName := ui.rooms[msg.Room]
			ui.mutex.RUnlock()
			if roomName == "" {
				roomName = msg.Room
			}
			fmt.Printf("[%s] [Room: %s] %s: %s\n", timestamp, roomName, msg.Sender, msg.Content)
		} else if msg.Recipient == "*" || msg.Recipient == "" {
			// Broadcast. This test must come BEFORE the "our own message" one
			// below: our own broadcast comes back with Sender == our nickname, so
			// with the other order it was reported as "[Private -> *]".
			fmt.Printf("[%s] %s: %s\n", timestamp, msg.Sender, msg.Content)
		} else if msg.Recipient == me {
			// Private message addressed to us.
			fmt.Printf("[%s] [Private] %s: %s\n", timestamp, msg.Sender, msg.Content)
		} else if msg.Sender == me {
			// A copy of our own private message, which the server deliberately
			// echoes back to the sender. Without this branch Recipient pointed at
			// the other person, so no condition matched and "/msg bob hi" printed
			// nothing on our side.
			fmt.Printf("[%s] [Private -> %s] %s\n", timestamp, msg.Recipient, msg.Content)
		}

	case common.TypeUserList:
		// ui.users is read by showUsers() from the input goroutine and written here
		// from the receiving goroutine - it needs a lock, just like ui.rooms.
		ui.mutex.Lock()
		ui.users = msg.Users
		ui.mutex.Unlock()

	case common.TypeStatus:
		fmt.Printf("[%s] %s changed status to %s\n", timestamp, msg.Sender, msg.Status)

	case common.TypeRoom:
		switch msg.Action {
		case common.RoomCreate:
			// msg.RoomName carries JUST the room name. We used to store msg.Content
			// here - the server's whole sentence ("Room 'x' created successfully") -
			// so "/room list" printed nonsense and, contrary to its own comment, the
			// map did not hold roomID -> roomName.
			ui.mutex.Lock()
			ui.rooms[msg.Room] = msg.RoomName
			ui.mutex.Unlock()
			fmt.Printf("[%s] %s (ID: %s)\n", timestamp, msg.Content, msg.Room)
		case common.RoomJoin:
			// Add room to our list when we join
			ui.mutex.Lock()
			ui.rooms[msg.Room] = msg.RoomName
			ui.mutex.Unlock()
			fmt.Printf("[%s] Joined room '%s' (ID: %s)\n", timestamp, msg.RoomName, msg.Room)
		case common.RoomMembers:
			// Display room members
			fmt.Printf("[%s] %s\n", timestamp, msg.Content)
		case common.RoomLeaveConfirm:
			// Remove room from local state after confirmation
			ui.mutex.Lock()
			delete(ui.rooms, msg.Room)
			ui.mutex.Unlock()
			// Content carries the full message (leave / kick / room deleted),
			// RoomName just the name - print the message, because only it
			// distinguishes the three cases.
			fmt.Printf("[%s] %s\n", timestamp, msg.Content)
		}

	case common.TypeInvite:
		fmt.Printf("\n[%s] %s\n", timestamp, msg.Content)
		fmt.Printf("Type '/room accept %s' to accept or '/room decline %s' to decline\n", msg.Room, msg.Room)

	case common.TypeFile:
		if msg.Sender == me && msg.RefID != "" {
			// The server accepted our upload and assigned its ID - start streaming.
			ui.fileTransfer.startOutgoing(msg.RefID, msg.FileID)
			return
		}
		fmt.Printf("[%s] %s is sending you file: %s (%s)\n",
			timestamp, msg.Sender, msg.Filename, formatFileSize(msg.Filesize))

	case common.TypeFileChunk:
		// Progress update
		fmt.Printf("\rFile transfer: %s - %s", msg.Filename, msg.Content)

	case common.TypeFileComplete:
		// The server sends FileComplete to BOTH sides. On the sender the transfer
		// record was already removed by notifyComplete, so trying to "receive"
		// ended with a misleading "File received" plus "file transfer not found".
		if !ui.fileTransfer.IsIncoming(msg.FileID) {
			fmt.Printf("\n[%s] File sent: %s\n", timestamp, msg.Filename)
			return
		}
		fmt.Printf("\n[%s] File received: %s\n", timestamp, msg.Filename)
		if path, err := ui.fileTransfer.ReceiveFile(msg.FileID); err != nil {
			fmt.Printf("Error saving file: %v\n", err)
		} else {
			fmt.Printf("File saved to %s\n", path)
		}

	case common.TypeError:
		if msg.RefID != "" {
			// The server rejected one of our uploads - drop the pending record.
			ui.fileTransfer.abortPending(msg.RefID)
		}
		fmt.Printf("[%s] Error: %s\n", timestamp, msg.Error)

	default:
		// System messages
		if msg.Sender == "Server" {
			fmt.Printf("[%s] %s\n", timestamp, msg.Content)
		}
	}
}

// showUsers displays online users
func (ui *UI) showUsers() {
	ui.mutex.RLock()
	users := append([]string(nil), ui.users...)
	ui.mutex.RUnlock()

	fmt.Println("\n=== Online Users ===")
	for _, user := range users {
		parts := strings.Split(user, ":")
		if len(parts) == 2 {
			fmt.Printf("  %s (%s)\n", parts[0], parts[1])
		} else {
			fmt.Printf("  %s\n", user)
		}
	}
	fmt.Printf("==================\n\n")
}

// showRooms displays user's rooms
func (ui *UI) showRooms() {
	fmt.Println("\n=== Your Rooms ===")
	ui.mutex.RLock()
	defer ui.mutex.RUnlock()
	if len(ui.rooms) == 0 {
		fmt.Println("  No rooms joined")
	} else {
		for id, info := range ui.rooms {
			fmt.Printf("  %s: %s\n", id, info)
		}
	}
	fmt.Printf("==================\n\n")
}

// showTransfers displays active file transfers
func (ui *UI) showTransfers() {
	transfers := ui.fileTransfer.TransferProgress()

	fmt.Println("\n=== File Transfers ===")
	if len(transfers) == 0 {
		fmt.Println("  No active transfers")
	} else {
		for _, transfer := range transfers {
			fmt.Printf("  %s\n", transfer)
		}
	}
	fmt.Printf("===================\n\n")
}
