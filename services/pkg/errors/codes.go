// Package errors defines unified API error codes for VoxMesh.
package errors

import "fmt"

// Error code ranges:
// 40xxx — Authentication
// 41xxx — Channel
// 42xxx — Gateway / Device
// 43xxx — Rate limiting
// 50xxx — Server

const (
	// Auth errors (40xxx)
	CodeInvalidCredentials  = 40001
	CodeTokenExpired        = 40002
	CodeTokenRevoked        = 40003
	CodeInvalidToken        = 40004
	CodeEmailTaken          = 40005
	CodeUsernameTaken       = 40006
	CodeInvalidRefreshToken = 40007

	// Channel errors (41xxx)
	CodeChannelNotFound    = 41001
	CodeChannelFull        = 41002
	CodeChannelLocked      = 41003 // password required / wrong
	CodeAlreadyInChannel   = 41004
	CodeNotInChannel       = 41005
	CodeCannotCreateChild  = 41006
	CodeChannelLimitReached = 41007

	// Gateway / Device errors (42xxx)
	CodeGatewayNotFound     = 42001
	CodeGatewayOffline      = 42002
	CodeDeviceNotFound      = 42003
	CodeDeviceUnreachable   = 42004
	CodeInvalidGatewayKey   = 42005
	CodeGatewayAlreadyExists = 42006
	CodeNoFailoverGateway   = 42007

	// Rate limiting (43xxx)
	CodeTooManyRequests   = 43001
	CodeTooManyChannels   = 43002

	// Server errors (50xxx)
	CodeInternalError    = 50001
	CodeDatabaseError    = 50002
	CodeMQTTError        = 50003
	CodeServiceUnavailable = 50004
)

// APIError represents a structured API error response.
type APIError struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func New(code int, message string) *APIError {
	return &APIError{Code: code, Message: message}
}

func NewWithDetails(code int, message string, details map[string]any) *APIError {
	return &APIError{Code: code, Message: message, Details: details}
}

// Predefined errors for common cases.
var (
	ErrInvalidCredentials  = New(CodeInvalidCredentials, "invalid email or password")
	ErrTokenExpired        = New(CodeTokenExpired, "access token has expired")
	ErrTokenRevoked        = New(CodeTokenRevoked, "refresh token has been revoked")
	ErrInvalidToken        = New(CodeInvalidToken, "invalid access token")
	ErrInvalidRefreshToken = New(CodeInvalidRefreshToken, "invalid refresh token")
	ErrEmailTaken          = New(CodeEmailTaken, "email already registered")
	ErrUsernameTaken       = New(CodeUsernameTaken, "username already taken")
	ErrChannelNotFound     = New(CodeChannelNotFound, "channel not found")
	ErrChannelFull         = New(CodeChannelFull, "channel is full")
	ErrChannelLocked       = New(CodeChannelLocked, "channel requires a password")
	ErrAlreadyInChannel    = New(CodeAlreadyInChannel, "already in this channel")
	ErrNotInChannel        = New(CodeNotInChannel, "not currently in this channel")
	ErrCannotCreateChild   = New(CodeCannotCreateChild, "cannot operate on channel with children")
	ErrInternal            = New(CodeInternalError, "internal server error")
	ErrServiceUnavailable  = New(CodeServiceUnavailable, "service unavailable")
)
