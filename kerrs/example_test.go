package kerrs_test

import (
	"errors"
	"fmt"
	"strings"

	"go/build"

	"bytes"

	"github.com/theplant/appkit/kerrs"
	"github.com/theplant/testingutils"
)

func ExampleWrapv_errors() {
	err0 := errors.New("hi, I am an error")
	err1 := kerrs.Wrapv(err0, "wrong", "code", "12123", "value", 12312)

	// fmt.Printf("%+v", err)
	err2 := kerrs.Wrapv(err1, "more explain about the error", "morecontext", "999")

	actual := cleanStacktrace(fmt.Sprintf("\n%+v\n", err2))
	expected := `
more explain about the error morecontext=999: wrong code=12123 value=12312: hi, I am an error
github.com/theplant/appkit/kerrs.Wrapv
	github.com/theplant/appkit/kerrs/errors.go:27
github.com/theplant/appkit/kerrs_test.ExampleWrapv_errors
	github.com/theplant/appkit/kerrs/example_test.go:21
testing.runExample
	testing/example.go:122
testing.runExamples
	testing/example.go:46
testing.(*M).Run
	testing/testing.go:823
main.main
	github.com/theplant/appkit/kerrs/_test/_testmain.go:52
runtime.main
	runtime/proc.go:185
runtime.goexit
	runtime/asm_amd64.s:2197
`
	diff := testingutils.PrettyJsonDiff(expected, actual)
	fmt.Println(diff)
	// Output:
	//

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
	github.com/theplant/appkit/kerrs/example_test.go:89
testing.runExample
	testing/example.go:122
testing.runExamples
	testing/example.go:46
testing.(*M).Run
	testing/testing.go:823
main.main
	github.com/theplant/appkit/kerrs/_test/_testmain.go:52
runtime.main
	runtime/proc.go:185
runtime.goexit
	runtime/asm_amd64.s:2197

more explain about the error product_name=iphone color=red: wrong code=12123 value=12312: hi, I am an error
github.com/theplant/appkit/kerrs.Wrapv
	github.com/theplant/appkit/kerrs/errors.go:27
github.com/theplant/appkit/kerrs_test.ExampleExtract_errors
	github.com/theplant/appkit/kerrs/example_test.go:88
testing.runExample
	testing/example.go:122
testing.runExamples
	testing/example.go:46
testing.(*M).Run
	testing/testing.go:823
main.main
	github.com/theplant/appkit/kerrs/_test/_testmain.go:52
runtime.main
	runtime/proc.go:185
runtime.goexit
	runtime/asm_amd64.s:2197

wrong code=12123 value=12312: hi, I am an error
github.com/theplant/appkit/kerrs.Wrapv
	github.com/theplant/appkit/kerrs/errors.go:27
github.com/theplant/appkit/kerrs_test.ExampleExtract_errors
	github.com/theplant/appkit/kerrs/example_test.go:87
testing.runExample
	testing/example.go:122
testing.runExamples
	testing/example.go:46
testing.(*M).Run
	testing/testing.go:823
main.main
	github.com/theplant/appkit/kerrs/_test/_testmain.go:52
runtime.main
	runtime/proc.go:185
runtime.goexit
	runtime/asm_amd64.s:2197`
	diff := testingutils.PrettyJsonDiff(expected, actual.String())
	fmt.Println(diff)
	// Output:
	//

}

func cleanStacktrace(stacktrace string) (cleantrace string) {
	cleantrace = strings.Replace(stacktrace, build.Default.GOPATH+"/src/", "", -1)
	cleantrace = strings.Replace(cleantrace, build.Default.GOROOT+"/src/", "", -1)
	return
}
