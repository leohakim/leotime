package apierr

type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Error struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
}

type Response struct {
	Error Error `json:"error"`
}

func Validation(field, code, message string) Error {
	return Error{
		Code:    "validation_failed",
		Message: message,
		Fields: []FieldError{{
			Field:   field,
			Code:    code,
			Message: message,
		}},
	}
}

func Simple(code, message string) Error {
	return Error{Code: code, Message: message}
}
