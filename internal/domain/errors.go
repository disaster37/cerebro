package domain

import "errors"

var (
	ErrNotFound     = errors.New("resource not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrInvalidInput = errors.New("invalid input")
)

type MissingRequiredParamError struct{ Field string }

func (e *MissingRequiredParamError) Error() string { return e.Field + " is required" }

// NOTE: currently unused (never constructed); kept for parity with process().
type RequestFailedError struct {
	Path   string
	Status int
	Body   string
}

func (e *RequestFailedError) Error() string { return "Failure for [" + e.Path + "]" }
