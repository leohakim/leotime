package mail

import "errors"

var ErrPermanent = errors.New("permanent mail error")

type PermanentError struct {
	Err error
}

func (e PermanentError) Error() string {
	if e.Err == nil {
		return ErrPermanent.Error()
	}
	return e.Err.Error()
}

func (e PermanentError) Unwrap() error {
	return e.Err
}

func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return PermanentError{Err: err}
}

type TransientError struct {
	Err error
}

func (e TransientError) Error() string {
	if e.Err == nil {
		return "transient mail error"
	}
	return e.Err.Error()
}

func (e TransientError) Unwrap() error {
	return e.Err
}

func Transient(err error) error {
	if err == nil {
		return nil
	}
	return TransientError{Err: err}
}

func IsPermanent(err error) bool {
	var permanent PermanentError
	return errors.As(err, &permanent)
}
