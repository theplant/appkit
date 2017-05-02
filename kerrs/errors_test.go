package kerrs_test

import (
	"fmt"

	"github.com/theplant/appkit/kerrs"
)

func ExampleNewv_errors() {
	err1 := kerrs.Newv("Hi, I am an error!", "code", "12123", "value", 12312)

	// fmt.Printf("%+v", err)
	err2 := kerrs.Wrapv(err1, "more explain about the error", "morecontext", "999")

	fmt.Printf("%+v\n\n", err2)

	err3 := kerrs.Append(err1, err2, err1)

	fmt.Printf("%+v\n", err3)

	// Output:
	// more explain about the error morecontext=999: Hi, I am an error! code=12123 value=12312
	// github.com/theplant/appkit/kerrs.Wrapv
	// 	/Users/sunfmin/gopkg/src/github.com/theplant/appkit/kerrs/errors.go:21
	// github.com/theplant/appkit/kerrs_test.ExampleNewv_errors
	// 	/Users/sunfmin/gopkg/src/github.com/theplant/appkit/kerrs/errors_test.go:13
	// testing.runExample
	// 	/usr/local/Cellar/go/1.8/libexec/src/testing/example.go:122
	// testing.runExamples
	// 	/usr/local/Cellar/go/1.8/libexec/src/testing/example.go:46
	// testing.(*M).Run
	// 	/usr/local/Cellar/go/1.8/libexec/src/testing/testing.go:823
	// main.main
	// 	github.com/theplant/appkit/kerrs/_test/_testmain.go:44
	// runtime.main
	// 	/usr/local/Cellar/go/1.8/libexec/src/runtime/proc.go:185
	// runtime.goexit
	// 	/usr/local/Cellar/go/1.8/libexec/src/runtime/asm_amd64.s:2197
	//
	// 3 errors occurred:
	//
	// * Hi, I am an error! code=12123 value=12312
	// * more explain about the error morecontext=999: Hi, I am an error! code=12123 value=12312
	// * Hi, I am an error! code=12123 value=12312

}
