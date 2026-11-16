package common

import (
	"errors"
	"fmt"
)

// ErrorType classifies a ChatError. The constants are named Kind* rather than
// Err*: by convention an Err-prefixed identifier is an error VALUE (a sentinel
// to compare with errors.Is), and these are plain strings.
type ErrorType string

const (
	KindValidation   ErrorType = "VALIDATION"
	KindRateLimit    ErrorType = "RATE_LIMIT"
	KindNotFound     ErrorType = "NOT_FOUND"
	KindUnauthorized ErrorType = "UNAUTHORIZED"
	KindInternal     ErrorType = "INTERNAL"
	KindTimeout      ErrorType = "TIMEOUT"
	KindDuplicate    ErrorType = "DUPLICATE"
)

// ChatError represents a custom error with context
type ChatError struct {
	Type    ErrorType
	Message string
	Details map[string]any
}

// Error implements the error interface
func (e *ChatError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// NewChatError creates a new chat error
func NewChatError(errType ErrorType, message string) *ChatError {
	return &ChatError{
		Type:    errType,
		Message: message,
		Details: make(map[string]any),
	}
}

// WithDetail adds a detail to the error
func (e *ChatError) WithDetail(key string, value any) *ChatError {
	e.Details[key] = value
	return e
}

// IsType reports whether err is a *ChatError of the given type.
//
// errors.As rather than a direct type assertion: an assertion only inspects the
// outermost error, so it returns false as soon as the *ChatError has been wrapped
// with fmt.Errorf("...: %w", err). errors.As walks the whole chain.
func IsType(err error, errType ErrorType) bool {
	var chatErr *ChatError
	if errors.As(err, &chatErr) {
		return chatErr.Type == errType
	}
	return false
}
