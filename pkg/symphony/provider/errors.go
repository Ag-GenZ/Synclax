package provider

import (
	"context"
	"errors"
)

var (
	ErrProviderNotFound    = errors.New("provider_not_found")
	ErrResponseTimeout     = errors.New("response_timeout")
	ErrTurnTimeout         = errors.New("turn_timeout")
	ErrProcessExit         = errors.New("process_exit")
	ErrResponseError       = errors.New("response_error")
	ErrTurnFailed          = errors.New("turn_failed")
	ErrTurnCancelled       = errors.New("turn_cancelled")
	ErrTurnInputRequired   = errors.New("turn_input_required")
	ErrUnsupportedToolCall = errors.New("unsupported_tool_call")
)

type Error struct {
	Category string
	Err      error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return e.Category
	}
	return e.Category + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error { return e.Err }

func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	var pe *Error
	if errors.As(err, &pe) && pe != nil {
		switch pe.Category {
		case ErrResponseTimeout.Error(), ErrTurnTimeout.Error():
			return true
		}
		// Fall back to wrapped error classification.
		if errors.Is(pe.Err, context.DeadlineExceeded) {
			return true
		}
	}
	return errors.Is(err, context.DeadlineExceeded)
}

