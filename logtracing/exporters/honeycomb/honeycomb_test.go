package honeycomb

import (
	"context"
	"testing"

	"github.com/honeycombio/libhoney-go"
	"github.com/honeycombio/libhoney-go/transmission"
	"github.com/theplant/appkit/logtracing"
)

func TestExporter(t *testing.T) {
	mockSender := &transmission.MockSender{}
	exporter, err := NewExporter(libhoney.Config{
		WriteKey:     "mock",
		Dataset:      "mock",
		Transmission: mockSender,
	})
	if err != nil {
		t.Fatal("new exporter initialization should be successful")
	}
	logtracing.RegisterExporter(exporter)

	ctx := context.Background()
	ctx, _ = logtracing.StartSpan(ctx, "test")
	logtracing.EndSpan(ctx, nil)
	exporter.Close()

	ev := mockSender.Events()[0]
	if ev == nil {
		t.Fatal("event should not be nil")
	}
	if ev.Data["span.id"] == nil {
		t.Fatal("span.id should not be nil")
	}
}
