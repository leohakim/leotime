package store

import (
	"errors"
	"testing"
)

func TestValidationErrorIsSentinel(t *testing.T) {
	err := validationError(ErrInvalidClientInput, "name", "required", "name is required")

	if !errors.Is(err, ErrInvalidClientInput) {
		t.Fatal("expected sentinel match")
	}

	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatal("expected validation error type")
	}
	if validation.Field != "name" || validation.Code != "required" {
		t.Fatalf("unexpected validation payload: %+v", validation)
	}
}

func TestIsValidation(t *testing.T) {
	err := validationError(ErrInvalidTaskInput, "name", "required", "name is required")
	if !IsValidation(err, ErrInvalidTaskInput) {
		t.Fatal("expected IsValidation true")
	}
	if IsValidation(err, ErrInvalidClientInput) {
		t.Fatal("expected IsValidation false for other sentinel")
	}
}
