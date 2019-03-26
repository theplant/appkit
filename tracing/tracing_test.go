package tracing

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	opentracing "github.com/opentracing/opentracing-go"
)

func TestSpan_Noop(t *testing.T) {
	ctx := context.Background()
	err := Span(ctx, "noop", func(_ context.Context, _ opentracing.Span) error {
		return Span(ctx, "noop inner", func(_ context.Context, _ opentracing.Span) error {
			return nil
		})
	})

	if err != nil {
		t.Fatalf("received non-nil error: %v", err)
	}
}

func TestSpan_Error(t *testing.T) {
	ctx := context.Background()
	expected := errors.New("error")

	err := Span(ctx, "error", func(_ context.Context, _ opentracing.Span) error {
		return Span(ctx, "error inner", func(_ context.Context, _ opentracing.Span) error {
			return expected
		})
	})

	if err != expected {
		t.Fatalf("received unexpected error: %v", err)
	}
}

func TestSpan_Panic(t *testing.T) {
	ctx := context.Background()
	expected := errors.New("error")

	defer func() {
		r := recover()
		err, ok := r.(error)
		if !ok {
			t.Fatalf("non-error recovered: %v", r)
		}

		// Span will wrap the error before panicking
		if errors.Cause(err) != expected {
			t.Fatalf("unexpected value recovered: %v", err)
		}
	}()

	_ = Span(ctx, "panic", func(_ context.Context, _ opentracing.Span) error {
		return Span(ctx, "panic inner", func(_ context.Context, _ opentracing.Span) error {
			panic(expected)
		})
	})
}
