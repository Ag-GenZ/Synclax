package codex

import "errors"

var (
	ErrCodexNotFound        = errors.New("codex_not_found")
	ErrResponseTimeout      = errors.New("response_timeout")
	ErrTurnTimeout          = errors.New("turn_timeout")
	ErrPortExit             = errors.New("port_exit")
	ErrResponseError        = errors.New("response_error")
	ErrTurnFailed           = errors.New("turn_failed")
	ErrTurnCancelled        = errors.New("turn_cancelled")
	ErrTurnInputRequired    = errors.New("turn_input_required")
	ErrUnsupportedToolCall  = errors.New("unsupported_tool_call")
	ErrMalformedProtocolMsg = errors.New("malformed")
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
