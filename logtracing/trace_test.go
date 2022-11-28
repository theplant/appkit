package logtracing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/theplant/appkit/log"
	"github.com/theplant/testingutils/fatalassert"
)

func BenchmarkTracing(b *testing.B) {
	RegisterExporter(&mockedExporter{})

	ctx := log.Context(context.Background(), log.Default())
	ctx = ContextWithKVs(ctx, "key", "value")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx, _ := StartSpan(ctx, "test")
		AppendSpanKVs(ctx,
			"key", "value",
		)
		EndSpan(ctx, nil)
	}
}

func BenchmarkSampling(b *testing.B) {
	ctx := log.Context(context.Background(), log.Default())
	ApplyConfig(Config{
		DefaultSampler: ProbabilitySampler(0.25),
	})
	var (
		sampledCount   int
		unsampledCount int
	)
	for i := 0; i < b.N; i++ {
		ctx, s := StartSpan(ctx, "test")
		if s.isSampled {
			sampledCount++
		} else {
			unsampledCount++
		}
		EndSpan(ctx, nil)
	}

	b.Logf("Sampled count: %v, unsampled count: %v\n", sampledCount, unsampledCount)
}

func TestStartSpanWithoutParent(t *testing.T) {
	ctx := context.Background()
	ctx, s := StartSpan(ctx, "top-level")

	if s == nil {
		t.Fatalf("span should not be nil")
	}
	if s.name != "top-level" {
		t.Fatalf("span context should be the same as the name")
	}
	if len(s.traceID) == 0 {
		t.Fatalf("trace id should not be blank")
	}
	if len(s.spanID) == 0 {
		t.Fatalf("span id should not be blank")
	}

	if s.parentSpanID.IsValid() {
		t.Fatalf("parent span should be zero")
	}

	sInCtx := SpanFromContext(ctx)
	if sInCtx == nil || sInCtx.spanID != s.spanID {
		t.Fatalf("span should be in new ctx")
	}
}

func TestStartSpanWithParent(t *testing.T) {
	ctx := context.Background()

	ctx, topLevelS := StartSpan(ctx, "top-level")
	ctx, secondLevelS := StartSpan(ctx, "second-level")

	if secondLevelS == nil {
		t.Fatalf("span should not be nil")
	}
	if secondLevelS.name != "second-level" {
		t.Fatalf("span context should be the same as the name")
	}
	if len(secondLevelS.traceID) == 0 {
		t.Fatalf("trace id should not be blank")
	}
	if len(secondLevelS.spanID) == 0 {
		t.Fatalf("span id should not be blank")
	}
	if secondLevelS.parentSpanID != topLevelS.spanID {
		t.Fatalf("parent span should be equal to parent")
	}

	sInCtx := SpanFromContext(ctx)
	if sInCtx == nil || sInCtx.spanID != secondLevelS.spanID {
		t.Fatalf("span should be in new ctx")
	}
}

func TestEndSpan(t *testing.T) {
	ctx := context.Background()
	ctx, s := StartSpan(ctx, "test")

	err := errors.New("test error")

	EndSpan(ctx, err)

	if s.err != err {
		t.Fatalf("span should record the err")
	}
	if s.endTime.IsZero() {
		t.Fatalf("span end time should not be zero")
	}
	if s.Duration() == 0 {
		t.Fatalf("span duration should be greater than 0")
	}
}

func TestRecordPanic(t *testing.T) {
	ctx := context.Background()
	err := errors.New("I'm the danger!")

	defer func() {
		recovered := recover()
		if recovered != err {
			t.Fatalf("should receive panic")
		}

		s := SpanFromContext(ctx)
		if s.panic != err {
			t.Fatalf("panic should be recorded in span")
		}
	}()

	func() {
		ctx, _ = StartSpan(ctx, "test")
		defer RecordPanic(ctx)

		panic(err)
	}()
}

type mockedExporter struct {
	LastSpanData *SpanData
}

func (e *mockedExporter) ExportSpan(sd *SpanData) {
	e.LastSpanData = sd
}

func TestExportSpan(t *testing.T) {
	exporter := &mockedExporter{}
	RegisterExporter(exporter)

	ctx := context.Background()
	ctx, s := StartSpan(ctx, "test")
	AppendSpanKVs(ctx, "k", "v1")

	err := errors.New("test error")

	EndSpan(ctx, err)

	exportedSpanData := exporter.LastSpanData
	if exportedSpanData == nil {
		t.Fatal("exported span data should not be nil")
	}
	if exportedSpanData.SpanID != s.spanID {
		t.Fatal("exported span ID should be equal to the original span")
	}
	exportedSpanData.Keyvals[1] = "v2"
	if s.keyvals[1] != "v1" {
		t.Fatal("the original span keyvals should not be modified")
	}
}

func TestTrace(t *testing.T) {

	ctx := context.Background()
	ctx = ContextWithKVs(ctx, "key", "value")
	ctx, span := StartSpan(ctx, "top-level")
	defer func() { EndSpan(ctx, nil) }()
	if len(span.keyvals) != 2 {
		t.Fatalf("span should have 2 keyvals, but got %v", len(span.keyvals))
	}

	ctx2 := ContextWithKVs(ctx, "key2", "value2")
	ctx2, span2 := StartSpan(ctx2, "second-level")
	AppendSpanKVs(ctx2, "second-level-only", "test")
	defer func() { EndSpan(ctx2, nil) }()
	if len(span2.keyvals) != 6 {
		t.Fatalf("span should have 6 keyvals, but got %v", len(span2.keyvals))
	}

	ctx3, span3 := StartSpan(ctx2, "third-level")
	AppendSpanKVs(ctx3, "third-level-only", "test")
	defer func() { EndSpan(ctx3, errors.New("third-level-failed")) }()
	if len(span3.keyvals) != 6 {
		t.Fatalf("span should have 6 keyvals, but got %v", len(span3.keyvals))
	}
}

func TestTraceWithDefaultNeverSampler(t *testing.T) {
	ApplyConfig(Config{
		DefaultSampler: NeverSample(),
	})
	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test")
	defer func() { EndSpan(ctx, nil) }()
	if span.isSampled {
		t.Fatalf("span should not be sampled")
	}
}

func TestTraceWithNeverSampler(t *testing.T) {
	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test", WithSampler(NeverSample()))
	defer func() { EndSpan(ctx, nil) }()
	if span.isSampled {
		t.Fatalf("span should not be sampled")
	}
}

func TestChildrenAreSampledAsParent(t *testing.T) {
	ApplyConfig(Config{
		DefaultSampler: NeverSample(),
	})
	ctx := context.Background()
	pctx, pspan := StartSpan(ctx, "parent", WithSampler(AlwaysSample()))
	defer func() { EndSpan(pctx, nil) }()
	if !pspan.isSampled {
		t.Fatalf("parent span should be sampled")
	}

	cctx, cspan := StartSpan(pctx, "chiled")
	defer func() { EndSpan(cctx, nil) }()
	if !cspan.isSampled {
		t.Fatalf("child span should be sampled")
	}
}

func TestWithStartTime(t *testing.T) {
	ti := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	_, s := StartSpan(context.Background(), "test", WithStartTime(ti))
	fatalassert.Equal(t, s.startTime, ti)
}
