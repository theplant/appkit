package honeycomb

import (
	"fmt"

	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/theplant/appkit/logtracing"
)

func NewExporter(config libhoney.Config) (*exporter, error) {
	libhoney.UserAgentAddition = "Honeycomb-logtracing-exporter"

	err := libhoney.Init(config)
	if err != nil {
		return nil, fmt.Errorf("libhoney init failed: %w", err)
	}
	builder := libhoney.NewBuilder()

	return &exporter{
		builder: builder,
	}, nil
}

type exporter struct {
	builder     *libhoney.Builder
	ServiceName string
}

func (e *exporter) Close() {
	libhoney.Close()
}

func (e *exporter) ExportSpan(sd *logtracing.SpanData) {
	if sd == nil {
		return
	}

	ev := e.builder.NewEvent()
	ev.Timestamp = sd.StartTime

	if e.ServiceName != "" {
		ev.AddField("service_name", e.ServiceName)
	}

	dur := sd.EndTime.Sub(sd.StartTime)

	ev.AddField("trace.id", sd.TraceID)
	ev.AddField("span.id", sd.SpanID)
	ev.AddField("span.context", sd.Name)
	ev.AddField("span.dur_ms", dur.Milliseconds())

	if sd.IsSampled {
		ev.AddField("span.is_sampled", 1)
	}

	if sd.ParentSpanID.IsValid() {
		ev.AddField("span.parent_id", sd.ParentSpanID)
	}

	if sd.Err != nil {
		ev.AddField("span.err", sd.Err)
		ev.AddField("span.err_type", errType(sd.Err))
		ev.AddField("span.with_err", 1)
	}

	if sd.Panic != nil {
		ev.AddField("span.panic", sd.Panic)
		ev.AddField("span.panic_type", errType(sd.Panic))
		ev.AddField("span.with_panic", 1)
		ev.AddField("span.with_err", 1)
	}

	ev.AddField("msg", fmt.Sprintf("%s -> dur: %v err: %s panic: %s", sd.Name, dur, sd.Err, sd.Panic))
	ev.SendPresampled()
}

type causer interface {
	Cause() error
}

func errType(err interface{}) string {
	if c, ok := err.(causer); ok {
		return fmt.Sprintf("%T (%T)", c.Cause(), err)
	}
	return fmt.Sprintf("%T", err)
}
