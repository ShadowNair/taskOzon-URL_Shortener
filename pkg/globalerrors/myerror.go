package globalerrors

import (
	"errors"
)

var (
	ErrNotFound          = errors.New("link not found")
	ErrShortCodeConflict = errors.New("short code conflict")
	ErrInvalidURL        = errors.New("invalid url")
	ErrInvalidShortCode  = errors.New("invalid short code")
)
