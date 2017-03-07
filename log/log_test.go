package log_test

import (
	"testing"

	"github.com/theplant/appkit/log"
)

func TestLog(t *testing.T) {
	l := log.Default()
	err := l.Crit().Log("msg", "hello")
	if err != nil {
		t.Error(err)
	}
}
