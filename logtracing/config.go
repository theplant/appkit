package logtracing

import (
	"sync"
	"sync/atomic"

	"github.com/theplant/appkit/log"
)

// Config represents the global tracing configuration.
type Config struct {
	DefaultLogger  *log.Logger
	DefaultSampler Sampler
	IDGenerator    IDGenerator
}

var configWriteMu sync.Mutex

func ApplyConfig(cfg Config) {
	configWriteMu.Lock()
	defer configWriteMu.Unlock()
	c := *config.Load().(*Config)
	if cfg.DefaultLogger != nil {
		c.DefaultLogger = cfg.DefaultLogger
	}
	if cfg.DefaultSampler != nil {
		c.DefaultSampler = cfg.DefaultSampler
	}
	if cfg.IDGenerator != nil {
		c.IDGenerator = cfg.IDGenerator
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
