package main

import (
	"fmt"
	"strings"
)

type wrappederr struct {
	wrapped []error
	prefix  string
}

func (w wrappederr) Error() string {
	messages := make([]string, len(w.wrapped))
	for i, err := range w.wrapped {
		messages[i] = err.Error()
	}
	return w.prefix + ": " + strings.Join(messages, "; ")
}

func WrapErr(err error, prefix string, a ...interface{}) error {
	return WrapErrs([]error{err}, prefix, a...)
}

// WrapErrs combines the given errors into one with prefix and formatting.
// Returns nil if no non-nil errors are given.
func WrapErrs(errs []error, prefix string, a ...interface{}) error {
	nonNilErrs := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			nonNilErrs = append(nonNilErrs, err)
		}
	}
	if len(nonNilErrs) == 0 {
		return nil
	}
	return wrappederr{wrapped: errs, prefix: fmt.Sprintf(prefix, a...)}
}
