package kerrs_test

import (
	"errors"
	"testing"

	"github.com/theplant/appkit/kerrs"
	"github.com/theplant/testingutils"
)

func TestOddContextValues(t *testing.T) {
	err := kerrs.Wrapv(errors.New("hi"), "wrap message", "code")
	if err.Error() != `wrap message code="<value-missing>": hi` {
		t.Error(err)
	}
}

func TestExtractNormalError(t *testing.T) {
	err := errors.New("hi my error")

	kvs, msg, stacktrace := kerrs.Extract(err)
	expected := []interface{}{nil, "hi my error", ""}
	diff := testingutils.PrettyJsonDiff(expected, []interface{}{kvs, msg, stacktrace})
	if len(diff) > 0 {
		t.Error(diff)
	}
}
