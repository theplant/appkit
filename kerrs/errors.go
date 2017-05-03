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

/*
Whenever an error returned from other libraries you imported, and you didn't handle the error, you should wrap it and return it to upper side. By wrapping it, includes stacktrace, and any context values, like your func parameters, So that when it gets logged, It reveal more contexts for developer to know where and what the problem is.
*/
func Wrapv(err error, message string, keyvals ...interface{}) error {
	return perrs.WithStack(jerrs.With(keyvals...).Wrap(err, message))
}

/*
It's MultiError, useful when say you are looping csv file lines for return orders. one of them have error, But you should continue to deal with next lines, But you want the function to return error.

```go
func HandleCSV(csvfile ...) (err error) {
	for {
		lineErr := handleLine(line)
		if err != nil {
			err = kerrs.Append(err, err)
			continue
		}

		// NOT
		// if err != nil {
		//	return
		// }
	}
}
```
*/
func Append(err error, errs ...error) error {
	return merr.Append(err, errs...).ErrorOrNil()
}
