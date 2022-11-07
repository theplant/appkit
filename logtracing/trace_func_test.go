package logtracing

import (
	"context"
	"errors"
	"testing"
)

func TestTraceFunc(t *testing.T) {
	var s *span
	var testErr = errors.New("test error")
	fn := func(ctx context.Context) error {
		s = SpanFromContext(ctx)

		return testErr
	}

	err := TraceFunc(context.Background(), "fn", fn)
	if err != testErr {
		t.Fatalf("err should be test error")
	}

	if s == nil {
		t.Fatalf("span should be in context")
	}

	if s.name != "fn" {
		t.Fatalf("span context should be the same as the name")
	}

	if s.endTime.IsZero() {
		t.Fatalf("end time should not be zero")
	}

	if s.err != testErr {
		t.Fatalf("err should be test error")
	}

	panicErr := errors.New("I'm the danger!")
	panicerFN := func(ctx context.Context) error {
		s = SpanFromContext(ctx)

		panic(panicErr)
	}

	defer func() {
		recovered := recover()
		if recovered != panicErr {
			t.Fatalf("should receive panic")
		}

		if s.panic != panicErr {
			t.Fatalf("panic should be recorded in span")
		}
	}()

	func() {
		err = TraceFunc(context.Background(), "panicerFN", panicerFN)
	}()
}
