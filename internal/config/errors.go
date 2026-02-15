package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidationErrors collects multiple config validation errors.
type ValidationErrors struct {
	Errs []error
}

// Error returns a numbered list of all validation errors.
func (ve *ValidationErrors) Error() string {
	if len(ve.Errs) == 1 {
		return ve.Errs[0].Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d config errors:", len(ve.Errs))
	for i, err := range ve.Errs {
		fmt.Fprintf(&b, "\n  %d. %s", i+1, err.Error())
	}
	return b.String()
}

// Unwrap returns the underlying errors for use with errors.As/errors.Is.
func (ve *ValidationErrors) Unwrap() []error {
	return ve.Errs
}

// FormatYAMLError extracts human-readable messages from a YAML TypeError,
// including line numbers when available.
func FormatYAMLError(err error) []string {
	te, ok := err.(*yaml.TypeError)
	if !ok {
		return []string{err.Error()}
	}
	return te.Errors
}
