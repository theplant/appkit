

Package kerrs provide the following purpose

- Wrap context values pair for structure logging
- Include stacktrace automatically
- Support multiple error return for continue a loop when error happens




* [Append](#append)
* [Wrapv](#wrapv)




## Append
``` go
func Append(err error, errs ...error) error
```
Append returns a multi error, useful when say you are looping csv file lines for return orders. one of them have error, But you should continue to deal with next lines, But you want the function to return error.

```go
func HandleCSV(csvfile ...) (err error) {


```go
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
```

}
```



## Wrapv
``` go
func Wrapv(err error, message string, keyvals ...interface{}) error
```
Wrapv should be invoked whenever an error returned from other libraries you imported, and you didn't handle the error, you should wrap it and return it to upper side. By wrapping it, includes stacktrace, and any context values, like your func parameters, So that when it gets logged, It reveal more contexts for developer to know where and what the problem is.






