package main

import (
	"fmt"
)

type WrappedError struct {
	wrapped error
	prefix  string
}

func (w WrappedError) Error() string {
	return w.prefix + w.wrapped.Error()
}

func WrapErr(err error, prefix string, a ...interface{}) WrappedError {
	return WrappedError{wrapped: err, prefix: fmt.Sprintf(prefix, a...)}
}
