package server

import (
	"testing"
	"time"

	"github.com/theplant/appkit/log"
)

// By default (empty Config) no timeouts are set, so the http.Server keeps its
// zero values and never times out. This preserves the previous behaviour for
// services that don't opt in.
func TestNewServerNoTimeoutsByDefault(t *testing.T) {
	s := newServer(Config{Addr: ":9800"}, log.NewNopLogger(), nil)

	if s.ReadTimeout != 0 {
		t.Errorf("ReadTimeout: want 0, got %v", s.ReadTimeout)
	}
	if s.ReadHeaderTimeout != 0 {
		t.Errorf("ReadHeaderTimeout: want 0, got %v", s.ReadHeaderTimeout)
	}
	if s.WriteTimeout != 0 {
		t.Errorf("WriteTimeout: want 0, got %v", s.WriteTimeout)
	}
	if s.IdleTimeout != 0 {
		t.Errorf("IdleTimeout: want 0, got %v", s.IdleTimeout)
	}
}

// Each timeout in Config must map to its own field on the http.Server. Using
// distinct values guards against copy-paste mistakes (e.g. ReadHeaderTimeout
// accidentally taking ReadTimeout's value).
func TestNewServerMapsTimeouts(t *testing.T) {
	cfg := Config{
		Addr:              ":9800",
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	s := newServer(cfg, log.NewNopLogger(), nil)

	if s.ReadTimeout != cfg.ReadTimeout {
		t.Errorf("ReadTimeout: want %v, got %v", cfg.ReadTimeout, s.ReadTimeout)
	}
	if s.ReadHeaderTimeout != cfg.ReadHeaderTimeout {
		t.Errorf("ReadHeaderTimeout: want %v, got %v", cfg.ReadHeaderTimeout, s.ReadHeaderTimeout)
	}
	if s.WriteTimeout != cfg.WriteTimeout {
		t.Errorf("WriteTimeout: want %v, got %v", cfg.WriteTimeout, s.WriteTimeout)
	}
	if s.IdleTimeout != cfg.IdleTimeout {
		t.Errorf("IdleTimeout: want %v, got %v", cfg.IdleTimeout, s.IdleTimeout)
	}
}
