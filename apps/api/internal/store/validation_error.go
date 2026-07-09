package store

import "errors"

type ValidationError struct {
	Cause error
	Field string
	Code  string
	Msg   string
}

func (e *ValidationError) Error() string {
	return e.Msg
}

func (e *ValidationError) Is(target error) bool {
	return errors.Is(e.Cause, target)
}

func validationError(cause error, field, code, message string) error {
	return &ValidationError{Cause: cause, Field: field, Code: code, Msg: message}
}

func InvoiceInputError(field, code, message string) error {
	return validationError(ErrInvalidInvoiceInput, field, code, message)
}

func IsValidation(err, sentinel error) bool {
	var validation *ValidationError
	if errors.As(err, &validation) {
		return errors.Is(validation.Cause, sentinel)
	}
	return errors.Is(err, sentinel)
}
