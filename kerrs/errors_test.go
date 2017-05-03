package kerrs_test

import (
	"errors"
	"fmt"
	"strings"

	"github.com/theplant/appkit/kerrs"
)

func ExampleWrapv_errors() {
	err0 := errors.New("hi, I am an error")
	err1 := kerrs.Wrapv(err0, "wrong", "code", "12123", "value", 12312)

	// fmt.Printf("%+v", err)
	err2 := kerrs.Wrapv(err1, "more explain about the error", "morecontext", "999")

	fmt.Printf("%+v\n\n", err2)

	// Output:
	// more explain about the error morecontext=999: wrong code=12123 value=12312: hi, I am an error
	// github.com/theplant/appkit/kerrs.Wrapv
	// 	/Users/sunfmin/gopkg/src/github.com/theplant/appkit/kerrs/errors.go:22
	// github.com/theplant/appkit/kerrs_test.ExampleWrapv_errors
	// 	/Users/sunfmin/gopkg/src/github.com/theplant/appkit/kerrs/errors_test.go:16
	// testing.runExample
	// 	/usr/local/Cellar/go/1.8/libexec/src/testing/example.go:122
	// testing.runExamples
	// 	/usr/local/Cellar/go/1.8/libexec/src/testing/example.go:46
	// testing.(*M).Run
	// 	/usr/local/Cellar/go/1.8/libexec/src/testing/testing.go:823
	// main.main
	// 	github.com/theplant/appkit/kerrs/_test/_testmain.go:48
	// runtime.main
	// 	/usr/local/Cellar/go/1.8/libexec/src/runtime/proc.go:185
	// runtime.goexit
	// 	/usr/local/Cellar/go/1.8/libexec/src/runtime/asm_amd64.s:2197

}

func ExampleAppend_errors() {

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
}

func ExampleExtract_errors() {
	err0 := errors.New("hi, I am an error")
	err1 := kerrs.Wrapv(err0, "wrong", "code", "12123", "value", 12312)
	err2 := kerrs.Wrapv(err1, "more explain about the error", "product_name", "iphone", "color", "red")
	err3 := kerrs.Wrapv(err2, "in regexp", "request_id", "T1212123129983")
	kvs, msg, stacktrace := kerrs.Extract(err3)
	fmt.Println("msg:", msg)
	fmt.Printf("\nkeyvals:%#+v\n\n", kvs)
	fmt.Printf("stacktrace:\n%s", stacktrace)

	// Output:
	// msg: in regexp => more explain about the error => wrong => hi, I am an error
	//
	// keyvals:[]interface {}{"request_id", "T1212123129983", "product_name", "iphone", "color", "red", "code", "12123", "value", 12312}
	//
	// stacktrace:
	// in regexp request_id=T1212123129983: more explain about the error product_name=iphone color=red: wrong code=12123 value=12312: hi, I am an error
	// github.com/theplant/appkit/kerrs.Wrapv
	// 	/Users/sunfmin/gopkg/src/github.com/theplant/appkit/kerrs/errors.go:22
	// github.com/theplant/appkit/kerrs_test.ExampleExtract_errors
	// 	/Users/sunfmin/gopkg/src/github.com/theplant/appkit/kerrs/errors_test.go:80
	// testing.runExample
	// 	/usr/local/Cellar/go/1.8/libexec/src/testing/example.go:122
	// testing.runExamples
	// 	/usr/local/Cellar/go/1.8/libexec/src/testing/example.go:46
	// testing.(*M).Run
	// 	/usr/local/Cellar/go/1.8/libexec/src/testing/testing.go:823
	// main.main
	// 	github.com/theplant/appkit/kerrs/_test/_testmain.go:48
	// runtime.main
	// 	/usr/local/Cellar/go/1.8/libexec/src/runtime/proc.go:185
	// runtime.goexit
	// 	/usr/local/Cellar/go/1.8/libexec/src/runtime/asm_amd64.s:2197
	//
	// more explain about the error product_name=iphone color=red: wrong code=12123 value=12312: hi, I am an error
	// github.com/theplant/appkit/kerrs.Wrapv
	// 	/Users/sunfmin/gopkg/src/github.com/theplant/appkit/kerrs/errors.go:22
	// github.com/theplant/appkit/kerrs_test.ExampleExtract_errors
	// 	/Users/sunfmin/gopkg/src/github.com/theplant/appkit/kerrs/errors_test.go:79
	// testing.runExample
	// 	/usr/local/Cellar/go/1.8/libexec/src/testing/example.go:122
	// testing.runExamples
	// 	/usr/local/Cellar/go/1.8/libexec/src/testing/example.go:46
	// testing.(*M).Run
	// 	/usr/local/Cellar/go/1.8/libexec/src/testing/testing.go:823
	// main.main
	// 	github.com/theplant/appkit/kerrs/_test/_testmain.go:48
	// runtime.main
	// 	/usr/local/Cellar/go/1.8/libexec/src/runtime/proc.go:185
	// runtime.goexit
	// 	/usr/local/Cellar/go/1.8/libexec/src/runtime/asm_amd64.s:2197
	//
	// wrong code=12123 value=12312: hi, I am an error
	// github.com/theplant/appkit/kerrs.Wrapv
	// 	/Users/sunfmin/gopkg/src/github.com/theplant/appkit/kerrs/errors.go:22
	// github.com/theplant/appkit/kerrs_test.ExampleExtract_errors
	// 	/Users/sunfmin/gopkg/src/github.com/theplant/appkit/kerrs/errors_test.go:78
	// testing.runExample
	// 	/usr/local/Cellar/go/1.8/libexec/src/testing/example.go:122
	// testing.runExamples
	// 	/usr/local/Cellar/go/1.8/libexec/src/testing/example.go:46
	// testing.(*M).Run
	// 	/usr/local/Cellar/go/1.8/libexec/src/testing/testing.go:823
	// main.main
	// 	github.com/theplant/appkit/kerrs/_test/_testmain.go:48
	// runtime.main
	// 	/usr/local/Cellar/go/1.8/libexec/src/runtime/proc.go:185
	// runtime.goexit
	// 	/usr/local/Cellar/go/1.8/libexec/src/runtime/asm_amd64.s:2197
	//
	// hi, I am an error

}
