package logtracing

import (
	"sync"
	"sync/atomic"
	"time"
)

type Exporter interface {
	ExportSpan(s *SpanData)
}

type exportersMap map[Exporter]struct{}

var (
	exporterMu sync.Mutex
	exporters  atomic.Value
)

// RegisterExporter adds to the list of Exporters that will receive sampled
// trace spans.
func RegisterExporter(e Exporter) {
	exporterMu.Lock()
	new := make(exportersMap)
	if old, ok := exporters.Load().(exportersMap); ok {
		for k, v := range old {
			new[k] = v
		}
	}
	new[e] = struct{}{}
	exporters.Store(new)
	exporterMu.Unlock()
}

// UnregisterExporter removes from the list of Exporters the Exporter that was
// registered with the given name.
func UnregisterExporter(e Exporter) {
	exporterMu.Lock()
	new := make(exportersMap)
	if old, ok := exporters.Load().(exportersMap); ok {
		for k, v := range old {
			new[k] = v
		}
	}
	delete(new, e)
	exporters.Store(new)
	exporterMu.Unlock()
}

// SpanData contains all the information collected by a Span.
type SpanData struct {
	ParentSpanID SpanID

	TraceID
	SpanID
	Name      string
	IsSampled bool

	StartTime time.Time
	EndTime   time.Time

	Err   error
	Panic interface{}

	Keyvals []interface{}
}
