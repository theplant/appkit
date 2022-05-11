package logtracing

import (
	"context"
	"errors"
	"testing"

	"github.com/theplant/appkit/log"
)

func BenchmarkTracing(b *testing.B) {
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

	if s.parent != nil {
		t.Fatalf("parent span should be nil ")
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
	if secondLevelS.parent != topLevelS {
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
