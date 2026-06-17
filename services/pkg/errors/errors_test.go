package errors

import (
	"testing"
)

func TestNew(t *testing.T) {
	e := New(CodeChannelFull, "channel is full")
	if e.Code != CodeChannelFull {
		t.Errorf("expected code %d, got %d", CodeChannelFull, e.Code)
	}
	if e.Message != "channel is full" {
		t.Errorf("expected message, got %s", e.Message)
	}
}

func TestNewWithDetails(t *testing.T) {
	e := NewWithDetails(CodeTooManyRequests, "rate limited", map[string]any{"retry_after": 30})
	if e.Details["retry_after"] != 30 {
		t.Errorf("expected retry_after detail")
	}
}

func TestErrorString(t *testing.T) {
	e := New(CodeInternalError, "internal server error")
	if e.Error() != "[50001] internal server error" {
		t.Errorf("unexpected Error(): %s", e.Error())
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		err  *APIError
		code int
	}{
		{ErrInvalidCredentials, CodeInvalidCredentials},
		{ErrTokenExpired, CodeTokenExpired},
		{ErrInvalidToken, CodeInvalidToken},
		{ErrChannelNotFound, CodeChannelNotFound},
		{ErrChannelFull, CodeChannelFull},
		{ErrChannelLocked, CodeChannelLocked},
		{ErrNotInChannel, CodeNotInChannel},
		{ErrInternal, CodeInternalError},
		{ErrServiceUnavailable, CodeServiceUnavailable},
	}
	for _, tt := range tests {
		if tt.err.Code != tt.code {
			t.Errorf("expected code %d, got %d", tt.code, tt.err.Code)
		}
	}
}

func TestErrorCodeRanges(t *testing.T) {
	// Verify ranges don't overlap
	if CodeInvalidCredentials < 40000 || CodeInvalidCredentials >= 41000 {
		t.Errorf("auth errors out of 40xxx range")
	}
	if CodeChannelNotFound < 41000 || CodeChannelNotFound >= 42000 {
		t.Errorf("channel errors out of 41xxx range")
	}
	if CodeGatewayNotFound < 42000 || CodeGatewayNotFound >= 43000 {
		t.Errorf("gateway errors out of 42xxx range")
	}
	if CodeTooManyRequests < 43000 || CodeTooManyRequests >= 44000 {
		t.Errorf("rate limit errors out of 43xxx range")
	}
	if CodeInternalError < 50000 || CodeInternalError >= 51000 {
		t.Errorf("server errors out of 50xxx range")
	}
}
