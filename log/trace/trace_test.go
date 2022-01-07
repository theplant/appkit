package trace

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/theplant/appkit/log"
)

func BenchmarkTracing(b *testing.B) {
	ctx := log.Context(context.Background(), log.Default())
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx, s := StartSpan(ctx, "test")
		s.AddInheritableAttributes(
			Attribute("type", "test"),
		)
		s.AddAttributes(
			Attribute("key", "value"),
		)
		EndSpan(ctx, s, nil)
	}
}

func TestStartSpanWithoutParent(t *testing.T) {
	ctx := context.Background()
	ctx, s := StartSpan(ctx, "top-level")

	if s == nil {
		t.Fatalf("span should not be nil")
	}
	if s.spanContext != "top-level" {
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

	sInCtx := fromContext(ctx)
	if sInCtx == nil || sInCtx.spanID != s.spanID {
		t.Fatalf("span should be in new ctx")
	}
}

func TestStartSpanWithParent(t *testing.T) {
	ctx := context.Background()
	ctx, topLevelS := StartSpan(ctx, "top-level")
	topLevelS.AddInheritableAttributes(Attribute("family.name", "W"))

	ctx, secondLevelS := StartSpan(ctx, "second-level")

	if secondLevelS == nil {
		t.Fatalf("span should not be nil")
	}
	if secondLevelS.spanContext != "second-level" {
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
	if secondLevelS.inheritableAttributes["family.name"] != "W" {
		t.Fatalf("span should inherite specific attributes form parent")
	}

	sInCtx := fromContext(ctx)
	if sInCtx == nil || sInCtx.spanID != secondLevelS.spanID {
		t.Fatalf("span should be in new ctx")
	}
}

func TestEndSpan(t *testing.T) {
	ctx := context.Background()
	ctx, s := StartSpan(ctx, "test")

	err := errors.New("test error")

	EndSpan(ctx, s, err)

	if s.err != err {
		t.Fatalf("span should record the err")
	}
	if s.endTime == nil {
		t.Fatalf("span end time should not be nil")
	}
	if s.Duration() == 0 {
		t.Fatalf("span duration should be greater than 0")
	}
}

func TestInherableAttributes(t *testing.T) {
	s := span{}
	s.AddInheritableAttributes(Attribute("test_key", "test_value"))
	if s.inheritableAttributes["test_key"] != "test_value" {
		t.Fatalf("inheritable attribute should be added")
	}
}

func TestAddAttributes(t *testing.T) {
	s := span{}
	s.AddAttributes(
		Attribute("test_key", "test_value"),
	)
	if s.attributes["test_key"] != "test_value" {
		t.Fatalf("attribute should be added")
	}
}

func TestTrace(t *testing.T) {
	t.SkipNow()

	ctx := context.Background()
	ctx, span := StartSpan(ctx, "top-level")
	defer func() { EndSpan(ctx, span, nil) }()

	ctx2, span2 := StartSpan(ctx, "second-level")
	span2.AddAttributes(
		Attribute("second-level-only", "test"),
	)
	span2.AddInheritableAttributes(
		Attribute("second-level-inheritable", "test"),
		Attribute("second-level-inheritable-shoul-be-override", "test"),
	)
	time.Sleep(2 * time.Second)
	defer func() { EndSpan(ctx2, span2, nil) }()

	ctx3, span3 := StartSpan(ctx2, "third-level")
	span3.AddAttributes(
		Attribute("second-level-inheritable-shoul-be-override", "override"),
	)
	time.Sleep(3 * time.Second)
	defer func() { EndSpan(ctx3, span3, errors.New("third-level-failed")) }()
}
