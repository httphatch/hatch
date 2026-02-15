package config

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestValidationErrors_SingleError(t *testing.T) {
	ve := &ValidationErrors{Errs: []error{fmt.Errorf("version must be 1")}}
	got := ve.Error()
	want := "version must be 1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestValidationErrors_MultipleErrors(t *testing.T) {
	ve := &ValidationErrors{Errs: []error{
		fmt.Errorf("version must be 1"),
		fmt.Errorf("settings.tld must be one of: test, localhost, local, dev"),
	}}
	got := ve.Error()
	if !strings.HasPrefix(got, "2 config errors:") {
		t.Errorf("expected prefix '2 config errors:', got %q", got)
	}
	if !strings.Contains(got, "1. version must be 1") {
		t.Errorf("expected numbered first error, got %q", got)
	}
	if !strings.Contains(got, "2. settings.tld") {
		t.Errorf("expected numbered second error, got %q", got)
	}
}

func TestValidationErrors_ErrorsAs(t *testing.T) {
	ve := &ValidationErrors{Errs: []error{fmt.Errorf("bad config")}}
	wrapped := fmt.Errorf("invalid config: %w", ve)

	var target *ValidationErrors
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As should match *ValidationErrors")
	}
	if len(target.Errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(target.Errs))
	}
}

func TestValidationErrors_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	ve := &ValidationErrors{Errs: []error{inner}}
	unwrapped := ve.Unwrap()
	if len(unwrapped) != 1 || unwrapped[0] != inner {
		t.Errorf("Unwrap returned unexpected errors: %v", unwrapped)
	}
}

func TestFormatYAMLError_TypeError(t *testing.T) {
	// Trigger an actual yaml.TypeError by unmarshaling invalid data.
	var cfg struct {
		Port int `yaml:"port"`
	}
	err := yaml.Unmarshal([]byte("port: not_a_number"), &cfg)
	if err == nil {
		t.Fatal("expected yaml error")
	}

	msgs := FormatYAMLError(err)
	if len(msgs) == 0 {
		t.Fatal("expected at least one message")
	}
	if !strings.Contains(msgs[0], "line") {
		t.Errorf("expected message with line info, got %q", msgs[0])
	}
}

func TestFormatYAMLError_RegularError(t *testing.T) {
	err := fmt.Errorf("something broke")
	msgs := FormatYAMLError(err)
	if len(msgs) != 1 || msgs[0] != "something broke" {
		t.Errorf("unexpected: %v", msgs)
	}
}
