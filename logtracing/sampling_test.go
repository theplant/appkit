package logtracing

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/theplant/appkit/log"
)

// withConfig applies cfg for one test and restores the previous global config.
func withConfig(t *testing.T, cfg Config) {
	t.Helper()
	old := config.Load()
	t.Cleanup(func() { config.Store(old) })
	ApplyConfig(cfg)
}

func contextWithBufferLogger(buf *bytes.Buffer) context.Context {
	return log.Context(context.Background(), log.Logger{Logger: kitlog.NewLogfmtLogger(buf)})
}

func TestNoTailSamplerKeepsHeadDecision(t *testing.T) {
	withConfig(t, Config{DefaultSampler: AlwaysSample()})

	ctx, _ := StartSpan(context.Background(), "root")
	cctx, child := StartSpan(ctx, "child")
	EndSpan(cctx, nil)

	if !child.sampled() {
		t.Fatal("without a tail sampler, a child of a sampled root must stay sampled")
	}
}

func TestRootOrErrorSpans(t *testing.T) {
	withConfig(t, Config{DefaultSampler: AlwaysSample(), TailSampler: RootOrErrorSpans()})

	ctx, root := StartSpan(context.Background(), "root")

	cctx, child := StartSpan(ctx, "child")
	EndSpan(cctx, nil)
	if child.sampled() {
		t.Error("a successful child span should be dropped by the tail sampler")
	}

	ectx, errChild := StartSpan(ctx, "error child")
	EndSpan(ectx, errors.New("boom"))
	if !errChild.sampled() {
		t.Error("a failed child span must be kept so errors stay visible")
	}

	pctx, panicChild := StartSpan(ctx, "panic child")
	func() {
		defer func() { _ = recover() }()
		defer RecordPanic(pctx)
		panic("boom")
	}()
	EndSpan(pctx, nil)
	if !panicChild.sampled() {
		t.Error("a panicked child span must be kept so panics stay visible")
	}

	EndSpan(ctx, nil)
	if !root.sampled() {
		t.Error("a root span must be kept")
	}

	var parentID SpanID
	parentID[0] = 1
	ictx, byID := StartSpan(context.Background(), "parent by id", WithParentSpanID(parentID))
	EndSpan(ictx, nil)
	if !byID.sampled() {
		t.Error("a span whose parent is given only by ID (e.g. from a traceparent header) has no parent span in its context and must be kept")
	}
}

func TestTailSamplerCannotResampleUnsampledSpan(t *testing.T) {
	withConfig(t, Config{
		DefaultSampler: NeverSample(),
		TailSampler:    func(*SpanData) bool { return true },
	})

	ctx, s := StartSpan(context.Background(), "root")
	EndSpan(ctx, nil)

	if s.sampled() {
		t.Fatal("the tail sampler must not override a head decision to drop the span")
	}
}

func TestUnsampledSpanIsLoggedWithIsSampledZero(t *testing.T) {
	withConfig(t, Config{DefaultSampler: AlwaysSample(), TailSampler: RootOrErrorSpans()})

	var buf bytes.Buffer
	ctx, _ := StartSpan(contextWithBufferLogger(&buf), "root")
	cctx, _ := StartSpan(ctx, "child")
	EndSpan(cctx, nil)

	out := buf.String()
	if !strings.Contains(out, "span.context=child") || !strings.Contains(out, "span.is_sampled=0") {
		t.Fatalf("by default a dropped span is still logged, marked span.is_sampled=0; got %q", out)
	}
}

func TestTailDroppedSpanIsNotExported(t *testing.T) {
	withConfig(t, Config{DefaultSampler: AlwaysSample(), TailSampler: RootOrErrorSpans()})
	exporter := &mockedExporter{}
	RegisterExporter(exporter)
	t.Cleanup(func() { UnregisterExporter(exporter) })

	ctx, _ := StartSpan(context.Background(), "root")
	cctx, _ := StartSpan(ctx, "child")
	EndSpan(cctx, nil)

	if exporter.LastSpanData != nil {
		t.Fatalf("a span dropped by the tail sampler must not be exported, got %q", exporter.LastSpanData.Name)
	}
}

func TestChildOfTailDroppedSpanKeepsHeadDecision(t *testing.T) {
	withConfig(t, Config{DefaultSampler: AlwaysSample(), TailSampler: RootOrErrorSpans()})

	ctx, _ := StartSpan(context.Background(), "root")
	mctx, _ := StartSpan(ctx, "mid")
	EndSpan(mctx, nil) // dropped by the tail sampler

	// Work that outlives its parent span, e.g. a goroutine.
	cctx, late := StartSpan(mctx, "late child")
	EndSpan(cctx, errors.New("boom"))

	if !late.sampled() {
		t.Fatal("a failed child must be kept even when its parent was tail-dropped: children inherit the head decision, not the tail one")
	}
}

func TestHeadUnsampledSpanLogOutputUnchanged(t *testing.T) {
	withConfig(t, Config{DefaultSampler: NeverSample()})

	var buf bytes.Buffer
	ctx, _ := StartSpan(contextWithBufferLogger(&buf), "root")
	EndSpan(ctx, nil)

	if strings.Contains(buf.String(), "span.is_sampled") {
		t.Fatalf("without a tail sampler, head-unsampled spans must log as before (no span.is_sampled key); got %q", buf.String())
	}
}

func TestTailSamplerNoRaceWithConcurrentChildStart(t *testing.T) {
	withConfig(t, Config{DefaultSampler: AlwaysSample(), TailSampler: RootOrErrorSpans()})

	ctx, _ := StartSpan(context.Background(), "root")
	mctx, _ := StartSpan(ctx, "mid")
	done := make(chan struct{})
	go func() {
		defer close(done)
		cctx, _ := StartSpan(mctx, "child")
		EndSpan(cctx, nil)
	}()
	EndSpan(mctx, nil)
	<-done
}
