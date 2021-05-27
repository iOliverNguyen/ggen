package ggen

import "github.com/olvrng/ggen/errors"

func Errorf(err error, format string, args ...interface{}) error {
	return errors.Errorf(err, format, args...)
}

func Errors(msg string, errs []error) error {
	return errors.Errors(msg, errs)
}
