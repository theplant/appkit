package service

import (
	"os"
	"testing"
	"time"

	"github.com/theplant/appkit/log"
)

func TestEnvDuration(t *testing.T) {
	const name = "TEST_SERVER_READ_TIMEOUT"

	cases := []struct {
		desc string
		set  bool
		val  string
		want time.Duration
	}{
		// Unset => 0 => timeout stays disabled (opt-in behaviour).
		{desc: "unset", set: false, want: 0},
		// Empty is treated the same as unset.
		{desc: "empty", set: true, val: "", want: 0},
		{desc: "valid seconds", set: true, val: "30s", want: 30 * time.Second},
		{desc: "valid minutes", set: true, val: "2m", want: 2 * time.Minute},
		{desc: "valid composite", set: true, val: "1m30s", want: 90 * time.Second},
		// Invalid => 0 (logged), never a wrong non-zero value that could kill
		// legitimate connections.
		{desc: "invalid garbage", set: true, val: "not-a-duration", want: 0},
		{desc: "invalid missing unit", set: true, val: "30", want: 0},
	}

	logger := log.NewNopLogger()

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			os.Unsetenv(name)
			if c.set {
				os.Setenv(name, c.val)
				defer os.Unsetenv(name)
			}

			if got := envDuration(logger, name); got != c.want {
				t.Errorf("envDuration(%q=%q): want %v, got %v", name, c.val, c.want, got)
			}
		})
	}
}
