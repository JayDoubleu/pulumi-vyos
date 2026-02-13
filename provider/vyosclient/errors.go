package vyosclient

import "fmt"

// APIError represents an error returned by the VyOS HTTP API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("vyos api error (status %d): %s", e.StatusCode, e.Message)
}

// AuthError indicates an authentication failure.
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("vyos auth error: %s", e.Message)
}
