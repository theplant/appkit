/*
This package provide the following purpose

- Wrap context values pair for structure logging
- Include stacktrace automatically
- Support multiple error return for continue a loop when error happens
*/
package kerrs

import (
	merr "github.com/hashicorp/go-multierror"
	jerrs "github.com/jjeffery/errors"
	perrs "github.com/pkg/errors"
)

func Newv(message string, keyvals ...interface{}) error {
	return perrs.WithStack(jerrs.With(keyvals...).New(message))
}

func Wrapv(err error, message string, keyvals ...interface{}) error {
	return perrs.WithStack(jerrs.With(keyvals...).Wrap(err, message))
}

func Append(err error, errs ...error) error {
	return merr.Append(err, errs...).ErrorOrNil()
}
