package service

import (
	"os"
	"testing"
	"time"

	"github.com/theplant/appkit/log"
)

// Each SERVER_* env var must map to its own timeout. Distinct values per var
// catch a mismapped field (e.g. SERVER_READ_TIMEOUT landing on WriteTimeout).
func TestServerTimeoutsMapping(t *testing.T) {
	envs := map[string]string{
		"SERVER_READ_HEADER_TIMEOUT": "1s",
		"SERVER_READ_TIMEOUT":        "2s",
		"SERVER_WRITE_TIMEOUT":       "3s",
		"SERVER_IDLE_TIMEOUT":        "4s",
	}
	for k, v := range envs {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	readHeader, read, write, idle := serverTimeouts(log.NewNopLogger())

	if readHeader != 1*time.Second {
		t.Errorf("readHeader: want 1s, got %v", readHeader)
	}
	if read != 2*time.Second {
		t.Errorf("read: want 2s, got %v", read)
	}
	if write != 3*time.Second {
		t.Errorf("write: want 3s, got %v", write)
	}
	if idle != 4*time.Second {
		t.Errorf("idle: want 4s, got %v", idle)
	}
}

// With nothing configured all timeouts stay disabled (0), so services that
// don't opt in are unaffected.
func TestServerTimeoutsUnsetAllZero(t *testing.T) {
	for _, k := range []string{
		"SERVER_READ_HEADER_TIMEOUT",
		"SERVER_READ_TIMEOUT",
		"SERVER_WRITE_TIMEOUT",
		"SERVER_IDLE_TIMEOUT",
	} {
		os.Unsetenv(k)
	}

	readHeader, read, write, idle := serverTimeouts(log.NewNopLogger())

	if readHeader != 0 || read != 0 || write != 0 || idle != 0 {
		t.Errorf("want all 0, got %v %v %v %v", readHeader, read, write, idle)
	}
}
