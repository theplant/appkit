package log_test

import (
	"context"
	"errors"
	"fmt"
	"go/build"
	"io"
	"strings"
	"testing"
	"time"

	"bytes"

	klog "github.com/go-kit/kit/log"

	stdl "log"

	"github.com/theplant/appkit/kerrs"
	"github.com/theplant/appkit/log"
	"github.com/theplant/testingutils"
)

func TestLog(t *testing.T) {
	l := log.Default()
	err := l.Crit().Log("msg", "hello")
	if err != nil {
		t.Error(err)
	}
}

var logErrCases = []struct {
	err      error
	expected string
}{
	{
		err: kerrs.Wrapv(io.EOF, "wrong io", "testcase", "TestLogError", "lineno", 23),
		expected: `
level error testcase TestLogError lineno 23 msg wrong io: EOF stacktrace wrong io testcase=TestLogError lineno=23: EOF
github.com/theplant/appkit/kerrs.Wrapv
	github.com/theplant/appkit/kerrs/errors.go:27
github.com/theplant/appkit/log_test.init`,
	},
	{
		err: errors.New("it's error"),
		expected: `
level error msg it's error
`,
	},
	{
		err: kerrs.Wrapv(io.EOF, "the message", "testcase", "TestLogError", "lineno"),
		expected: `
level error testcase TestLogError lineno <value-missing> msg the message: EOF stacktrace the message testcase=TestLogError lineno="<value-missing>": EOF
github.com/theplant/appkit/kerrs.Wrapv
	github.com/theplant/appkit/kerrs/errors.go:27
github.com/theplant/appkit/log_test.init`,
	},
}

func TestLogError(t *testing.T) {

	for _, errc := range logErrCases {
		output := bytes.NewBuffer(nil)
		output.WriteString("\n")
		l := log.Default()
		lev := klog.LoggerFunc(func(keyvals ...interface{}) (err error) {
			fmt.Fprintln(output, keyvals...)
			return nil
		})
		l = log.Logger{lev}

		l.WithError(errc.err).Log()
		diff := testingutils.PrettyJsonDiff(errc.expected, cleanStacktrace(output.String()))
		if len(diff) > 0 {
			t.Error(diff)
		}
	}
}

func cleanStacktrace(stacktrace string) (cleantrace string) {
	cleantrace = strings.Replace(stacktrace, build.Default.GOPATH+"/src/", "", -1)
	lines := strings.Split(cleantrace, "\n")
	if len(lines) >= 4 {
		lines = lines[0:5]
	}
	cleantrace = strings.Join(lines, "\n")
	return
}

func TestHuman(t *testing.T) {
	l := log.Default()
	err := l.WithError(kerrs.Wrapv(errors.New("original error"), "wrapped message", "code", 2000)).Log()
	if err != nil {
		t.Error(err)
	}

	log.SetStdLogOutput(l)

	l.WrapError(errors.New("hello error")).Log("msg", "there is a big error")

	stdl.Println("hello from go standard log")

	l.Info().Log("msg", "hello world", "order_code", "111222", "customer_id", "ABCDEFG")

	l.Debug().Log(
		"msg", fmt.Sprintf("auto-migrating %T", "table 1"),
		"table", "felix",
	)
	l.Info().Log(
		"msg", fmt.Sprintf("auto-migrating %T", "table 1"),
		"table", "felix",
	)
	l.Warn().Log(
		"msg", fmt.Sprintf("auto-migrating %T", "table 1"),
		"table", "felix",
	)
	l.Error().Log(
		"msg", fmt.Sprintf("auto-migrating %T", "table 1"),
		"table", "felix",
	)
	l.Crit().Log(
		"msg", fmt.Sprintf("auto-migrating %T", "table 1"),
		"table", "felix",
	)
}

var testContext = log.Context(context.TODO(), log.Default().With("app", "testapp"))

/*
log.Start will try to get logger from context, and log from there, every log with the instance
will print duration=301.356ms field with the log, the time the log time duration since log.Start

Example output:

```
15:16:20.34 hello app=testapp method=TestLogger store_id=100 duration=0.001ms
15:16:20.64 app=testapp method=TestLogger store_id=100 request_id=123 duration=300.542ms
15:16:20.64 debug app=testapp method=TestLogger store_id=100 duration=300.616ms
15:16:20.64 info app=testapp method=TestLogger store_id=100 duration=300.635ms
15:16:20.64 WrapError error: WrapError error app=testapp method=TestLogger store_id=100 duration=300.746ms
```
*/
func ExampleStart_log() {
	l := log.Start(testContext).With("method", "TestLogger", "store_id", 100)
	l.Log("msg", "hello")
	time.Sleep(100 * time.Millisecond)
	l.With("request_id", "123").Log()
	l.Debug().Log("msg", "debug")
	time.Sleep(200 * time.Millisecond)
	l.Info().Log("msg", "info")
	l.WrapError(errors.New("WrapError error")).Log()
	l.WithError(errors.New("WithError error")).Log()
	//Output:
}
