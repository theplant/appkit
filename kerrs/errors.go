/*
Package kerrs provide the following purpose

- Wrap context values pair for structure logging
- Include stacktrace automatically
- Support multiple error return for continue a loop when error happens
*/
package kerrs

import (
	"fmt"

	"strings"

	merr "github.com/hashicorp/go-multierror"
	jerrs "github.com/jjeffery/errors"
	perrs "github.com/pkg/errors"
)

/*
Wrapv should be invoked whenever an error returned from other libraries you imported, and you didn't handle the error, you should wrap it and return it to upper side. By wrapping it, includes stacktrace, and any context values, like your func parameters, So that when it gets logged, It reveal more contexts for developer to know where and what the problem is.
*/
func Wrapv(err error, message string, keyvals ...interface{}) error {
	if len(keyvals)%2 == 1 {
		keyvals = append(keyvals, "<value-missing>")
	}
	return perrs.WithStack(jerrs.With(keyvals...).Wrap(err, message))
}

/*
Append returns a multi error, useful when say you are looping csv file lines for return orders. one of them have error, But you should continue to deal with next lines, But you want the function to return error.

*/
func Append(err error, errs ...error) error {
	return merr.Append(err, errs...).ErrorOrNil()
}

/*
Extract an error of it's context values and message, it loop through to each level of errors, and concat each err message to a whole error message, and cause field is removed for easy to read, and concat each level error's stacktrace together to make a new whole stacktrace.
*/
func Extract(err error) (kvs []interface{}, msg string, stacktrace string) {
	if err == nil {
		return
	}

	var msgs []string
	var stacktraces []string
	type causer interface {
		Cause() error
	}

	type keyvaluer interface {
		Keyvals() []interface{}
	}

	var lastMsg interface{}

	for err != nil {
		lastMsg = err.Error()
		cause, isCauser := err.(causer)
		_, isKeyValuer := err.(keyvaluer)

		if !isKeyValuer && isCauser {
			stacktraces = append(stacktraces, fmt.Sprintf("%+v", err))
		}

		if !isCauser {
			break
		}
		err = cause.Cause()

		kver, isKeyValuer := err.(keyvaluer)
		if isKeyValuer {
			thekvs := kver.Keyvals()
			for i := 1; i < len(thekvs); i += 2 {
				key := thekvs[i-1]
				val := thekvs[i]
				if key == "msg" {
					msgs = append(msgs, fmt.Sprintf("%+v", val))
				} else if key == "cause" {
					lastMsg = val
				} else {
					kvs = append(kvs, key, val)
				}
			}
		}
	}
	msgs = append(msgs, fmt.Sprintf("%+v", lastMsg))
	msg = strings.Join(msgs, ": ")
	stacktrace = strings.Join(stacktraces, "\n\n")

	return
}
