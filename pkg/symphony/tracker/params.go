package tracker

import (
	"strconv"
	"strings"
)

// StringParam extracts a string value from a params map with a fallback default.
func StringParam(params map[string]any, key, fallback string) string {
	if params == nil {
		return fallback
	}
	v, ok := params[key]
	if !ok {
		return fallback
	}
	s, ok := v.(string)
	if !ok {
		return fallback
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	return s
}

// IntParam extracts an integer value from a params map with a fallback default.
func IntParam(params map[string]any, key string, fallback int) int {
	if params == nil {
		return fallback
	}
	v, ok := params[key]
	if !ok {
		return fallback
	}
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err == nil {
			return n
		}
	}
	return fallback
}

// BoolParam extracts a boolean value from a params map with a fallback default.
func BoolParam(params map[string]any, key string, fallback bool) bool {
	if params == nil {
		return fallback
	}
	v, ok := params[key]
	if !ok {
		return fallback
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "true", "1", "yes", "y", "on":
			return true
		case "false", "0", "no", "n", "off":
			return false
		}
	}
	return fallback
}
