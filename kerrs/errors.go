/*
Package kerrs provide the following purpose

- Wrap context values pair for structure logging
- Include stacktrace automatically
- Support multiple error return for continue a loop when error happens
*/
package kerrs

import (
	"fmt"

	merr "github.com/hashicorp/go-multierror"
	jerrs "github.com/jjeffery/errors"
	perrs "github.com/pkg/errors"
)

/*
Wrapv should be invoked whenever an error returned from other libraries you imported, and you didn't handle the error, you should wrap it and return it to upper side. By wrapping it, includes stacktrace, and any context values, like your func parameters, So that when it gets logged, It reveal more contexts for developer to know where and what the problem is.
*/
func Wrapv(err error, message string, keyvals ...interface{}) error {
	return perrs.WithStack(jerrs.With(keyvals...).Wrap(err, message))
}

/*
Append returns a multi error, useful when say you are looping csv file lines for return orders. one of them have error, But you should continue to deal with next lines, But you want the function to return error.

*/
func Append(err error, errs ...error) error {
	return merr.Append(err, errs...).ErrorOrNil()
}

/*
Extract an error of it's context values and message
*/
func Extract(err error) (kvs []interface{}, msg string, stacktrace string) {
	msg = ""
	stacktrace = ""
	type causer interface {
		Cause() error
	}

	type keyvaluer interface {
		Keyvals() []interface{}
	}

	var lastMsg interface{}

	for err != nil {
		_, ok := err.(keyvaluer)
		if !ok {
			stacktrace = stacktrace + fmt.Sprintf("%+v\n\n", err)
		}
		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()

		kver, ok := err.(keyvaluer)
		if ok {
			thekvs := kver.Keyvals()
			for i := 1; i < len(thekvs); i += 2 {
				key := thekvs[i-1]
				val := thekvs[i]
				if key == "msg" {
					if len(msg) == 0 {
						msg = fmt.Sprintf("%+v", val)
					} else {
						msg = fmt.Sprintf("%+v => %+v", msg, val)
					}
				} else if key == "cause" {
					lastMsg = val
				} else {
					kvs = append(kvs, key, val)
				}
			}
		}
	}
	msg = fmt.Sprintf("%+v => %+v", msg, lastMsg)

	return
}
