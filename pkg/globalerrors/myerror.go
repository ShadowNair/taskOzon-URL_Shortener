package globalerrors

import (
	"errors"
)

var (
	ErrNotFound = errors.New("link not found")
	ErrShortCodeConflict = errors.New("short code conflict")
)