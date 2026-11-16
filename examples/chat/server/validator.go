package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"training.pl/go/examples/chat/common"
)

var (
	nicknameRegex = regexp.MustCompile(common.NicknamePattern)
	roomNameRegex = regexp.MustCompile(common.RoomNamePattern)
)

// ValidateNickname validates a nickname according to the rules
func ValidateNickname(nickname string) error {
	// utf8.RuneCountInString, not len(): len counts BYTES, so a "20 characters"
	// limit would in practice mean ~6 CJK characters, or 10 accented Polish ones.
	length := utf8.RuneCountInString(nickname)
	if length < common.MinNicknameLength {
		return fmt.Errorf("nickname must be at least %d characters long", common.MinNicknameLength)
	}
	if length > common.MaxNicknameLength {
		return fmt.Errorf("nickname cannot exceed %d characters", common.MaxNicknameLength)
	}
	if !nicknameRegex.MatchString(nickname) {
		return errors.New("nickname can only contain letters, numbers, underscores, and hyphens")
	}
	// "Server" is the sender of every system message - without reserving it any
	// user could impersonate the server.
	if strings.EqualFold(nickname, "Server") {
		return errors.New("nickname 'Server' is reserved")
	}
	return nil
}

// ValidateRoomName validates a room name according to the rules.
// NOTE: TrimSpace works on a local copy only - we validate the same form the
// caller stores (handleRoomMessage trims as well).
func ValidateRoomName(roomName string) error {
	roomName = strings.TrimSpace(roomName)

	length := utf8.RuneCountInString(roomName)
	if length < common.MinRoomNameLength {
		return fmt.Errorf("room name must be at least %d characters long", common.MinRoomNameLength)
	}
	if length > common.MaxRoomNameLength {
		return fmt.Errorf("room name cannot exceed %d characters", common.MaxRoomNameLength)
	}
	if !roomNameRegex.MatchString(roomName) {
		return errors.New("room name can only contain letters, numbers, underscores, hyphens, and spaces")
	}
	return nil
}

// ValidateMessage validates a message content
func ValidateMessage(content string) error {
	if content == "" {
		return errors.New("message cannot be empty")
	}
	// MaxMessageSize is a limit in BYTES (it protects the read buffer), so the
	// message says bytes too - otherwise it would mislead for non-ASCII text.
	if len(content) > common.MaxMessageSize {
		return fmt.Errorf("message cannot exceed %d bytes", common.MaxMessageSize)
	}
	return nil
}

// ValidateFileName validates a file name for security
func ValidateFileName(filename string) error {
	if filename == "" {
		return errors.New("filename cannot be empty")
	}
	if len(filename) > common.MaxFileNameLength {
		return fmt.Errorf("filename cannot exceed %d characters", common.MaxFileNameLength)
	}

	// Check for path traversal attempts
	cleanPath := filepath.Clean(filename)
	if strings.Contains(cleanPath, "..") || strings.ContainsAny(cleanPath, `/\`) {
		return errors.New("filename cannot contain path separators or parent directory references")
	}

	// Check for hidden files
	if strings.HasPrefix(filename, ".") {
		return errors.New("hidden files are not allowed")
	}

	return nil
}

// ValidateFileSize validates file size is within limits
func ValidateFileSize(size int64) error {
	if size <= 0 {
		return errors.New("file size must be positive")
	}
	if size > common.MaxFileSize {
		return fmt.Errorf("file size cannot exceed %d bytes", common.MaxFileSize)
	}
	return nil
}
