package errors

import (
	"fmt"
	"runtime/debug"
)

type InternalError struct {
	Err      error  `json:"-"`
	IsPublic bool   `json:"-"`
	Message  string `json:"message"`
	Trace    string `json:"-"`
}

func (e *InternalError) Error() string {
	return e.Message
}

func (e *InternalError) Unwrap() error {
	return e.Err
}

func NewInternalError(err error, mes string, public bool) *InternalError {
	return &InternalError{
		Err:      err,
		IsPublic: public,
		Message:  mes,
		Trace:    fmt.Sprintf("%s\n%s", mes, debug.Stack()),
	}
}
