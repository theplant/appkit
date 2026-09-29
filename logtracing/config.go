package logtracing

import (
	"sync"
	"sync/atomic"
)

// Config represents the global tracing configuration.
type Config struct {
	DefaultSampler Sampler
	IDGenerator    IDGenerator

	// TailSampler, if set, is consulted when a span ends and can mark a
	// sampled span as unsampled. nil keeps the head sampling decision.
	TailSampler TailSampler
}

var configWriteMu sync.Mutex

// ApplyConfig applies the non-zero fields of cfg.
func ApplyConfig(cfg Config) {
	configWriteMu.Lock()
	defer configWriteMu.Unlock()
	c := *config.Load().(*Config)
	if cfg.DefaultSampler != nil {
		c.DefaultSampler = cfg.DefaultSampler
	}
	if cfg.IDGenerator != nil {
		c.IDGenerator = cfg.IDGenerator
	}
	if cfg.TailSampler != nil {
		c.TailSampler = cfg.TailSampler
	}
	config.Store(&c)
}

var config atomic.Value // access atomically

func init() {
	config.Store(&Config{
		DefaultSampler: AlwaysSample(),
		IDGenerator:    defaultIDGenerator(),
	})
}
