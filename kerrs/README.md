

Package kerrs provide the following purpose

- Wrap context values pair for structure logging
- Include stacktrace automatically
- Support multiple error return for continue a loop when error happens




* [Append](#append)
* [Extract](#extract)
* [Wrapv](#wrapv)




## Append
``` go
func Append(err error, errs ...error) error
```
Append returns a multi error, useful when say you are looping csv file lines for return orders. one of them have error, But you should continue to deal with next lines, But you want the function to return error.


```go
	var handleCSV = func(csvContent string) (err error) {
	    var handleLine = func(line string) (err error) {
	        if len(line) > 3 {
	            err = fmt.Errorf("Invalid Length for %s", line)
	        }
	        return
	    }
	    lines := strings.Split(csvContent, "\n")
	    for _, line := range lines {
	        lineErr := handleLine(line)
	        if lineErr != nil {
	            err = kerrs.Append(err, lineErr)
	            continue
	        }
	
	        // NOT
	        // if err != nil {
	        //	return
	        // }
	    }
	    return
	}
	
	err3 := handleCSV("a\n1234\nb11111\nc")
	fmt.Printf("%+v\n", err3)
	
	// Output:
	// 2 errors occurred:
	//
	// * Invalid Length for 1234
	// * Invalid Length for b11111
```

## Extract
``` go
func Extract(err error) (kvs []interface{}, msg string, stacktrace string)
```
Extract an error of it's context values and message, it loop through to each level of errors, and concat each err message to a whole error message, and cause field is removed for easy to read, and concat each level error's stacktrace together to make a new whole stacktrace.


```go
	err0 := errors.New("hi, I am an error")
	err1 := kerrs.Wrapv(err0, "wrong", "code", "12123", "value", 12312)
	err2 := kerrs.Wrapv(err1, "more explain about the error", "product_name", "iphone", "color", "red")
	err3 := kerrs.Wrapv(err2, "in regexp", "request_id", "T1212123129983")
	kvs, msg, stacktrace := kerrs.Extract(err3)
	
	var actual = bytes.NewBuffer(nil)
	fmt.Fprintln(actual, "\nmsg:", msg)
	fmt.Fprintf(actual, "\nkeyvals: %#+v\n\n", kvs)
	fmt.Fprintf(actual, "stacktrace:\n%s", cleanStacktrace(stacktrace))
	
	expected := `
	msg: in regexp: more explain about the error: wrong: hi, I am an error
	
	keyvals: []interface {}{"request_id", "T1212123129983", "product_name", "iphone", "color", "red", "code", "12123", "value", 12312}
	
	stacktrace:
	in regexp request_id=T1212123129983: more explain about the error product_name=iphone color=red: wrong code=12123 value=12312: hi, I am an error
	github.com/theplant/appkit/kerrs.Wrapv
	github.com/theplant/appkit/kerrs/errors.go:27
	github.com/theplant/appkit/kerrs_test.ExampleExtract_errors
	github.com/theplant/appkit/kerrs/example_test.go:76`
	
	diff := testingutils.PrettyJsonDiff(expected, actual.String())
	fmt.Println(diff)
	// Output:
	//
```

## Wrapv
``` go
func Wrapv(err error, message string, keyvals ...interface{}) error
```
Wrapv should be invoked whenever an error returned from other libraries you imported, and you didn't handle the error, you should wrap it and return it to upper side. By wrapping it, includes stacktrace, and any context values, like your func parameters, So that when it gets logged, It reveal more contexts for developer to know where and what the problem is.


```go
	err0 := errors.New("hi, I am an error")
	err1 := kerrs.Wrapv(err0, "wrong", "code", "12123", "value", 12312)
	
	// fmt.Printf("%+v", err)
	err2 := kerrs.Wrapv(err1, "more explain about the error", "morecontext", "999")
	
	actual := cleanStacktrace(fmt.Sprintf("\n%+v\n", err2))
	expected := `
	more explain about the error morecontext=999: wrong code=12123 value=12312: hi, I am an error
	github.com/theplant/appkit/kerrs.Wrapv
	github.com/theplant/appkit/kerrs/errors.go:27
	github.com/theplant/appkit/kerrs_test.ExampleWrapv_errors`
	
	diff := testingutils.PrettyJsonDiff(expected, actual)
	fmt.Println(diff)
	// Output:
	//
```




