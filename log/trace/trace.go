package trace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/theplant/appkit/log"
)

const (
	SpanTypeKey = "span.type"
	SpanRoleKey = "span.role"
)

func StartSpan(ctx context.Context, name string) (context.Context, *span) {
	var (
		parent = fromContext(ctx)

		traceID               string
		inheritableAttributes map[string]interface{}
	)

	if parent == nil {
		traceID = uuid.New().String()
	} else {
		traceID = parent.traceID
		if len(parent.inheritableAttributes) > 0 {
			inheritableAttributes = make(map[string]interface{})
			for k, v := range parent.inheritableAttributes {
				inheritableAttributes[k] = v
			}
		}
	}

	s := span{
		parent: parent,

		traceID:     traceID,
		spanID:      uuid.New().String(),
		spanContext: name,

		startTime: time.Now(),

		inheritableAttributes: inheritableAttributes,
	}

	return newContext(ctx, &s), &s
}

func EndSpan(ctx context.Context, s *span, err error) {
	s.recordError(err)
	s.end()

	logSpan(ctx, s)
}

func logSpan(ctx context.Context, s *span) {
	var (
		l        = log.ForceContext(ctx)
		keysvals []interface{}
		durMS    = s.Duration().Milliseconds()
	)

	keysvals = append(keysvals,
		"ts", s.startTime.Format(time.RFC3339Nano),
		"trace.id", s.traceID,
		"span.id", s.spanID,
		"span.context", s.spanContext,
		"span.dur_ms", durMS,
	)

	if s.parent != nil {
		keysvals = append(keysvals, "span.parent_id", s.parent.spanID)
	}

	for k, v := range s.inheritableAttributes {
		if _, ok := s.attributes[k]; ok {
			continue
		}
		keysvals = append(keysvals, k, v)
	}
	for k, v := range s.attributes {
		keysvals = append(keysvals, k, v)
	}

	if s.err != nil {
		keysvals = append(keysvals,
			"msg", fmt.Sprintf("%s (%v) -> %s (%T)", s.spanContext, durMS, s.err, s.err),
			"span.err", s.err,
			"span.err_type", errType(s.err),
			"span.with_err", 1,
		)
		l.Error().Log(keysvals...)
	} else if r := recover(); r != nil {
		keysvals = append(keysvals,
			"msg", fmt.Sprintf("%s (%v) -> panic: %s (%T)", s.spanContext, durMS, r, r),
			"span.err", r,
			"span.panic", 1,
			"span.err_type", errType(s.err),
			"span.with_err", 1,
		)
		l.Error().Log(keysvals...)
		panic(r)
	} else {
		keysvals = append(keysvals, "msg", fmt.Sprintf("%s (%v) -> success", s.spanContext, durMS))
		l.Info().Log(keysvals...)
	}
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

type contextKey struct{}

func fromContext(ctx context.Context) *span {
	s, _ := ctx.Value(contextKey{}).(*span)
	return s
}

func newContext(parent context.Context, s *span) context.Context {
	return context.WithValue(parent, contextKey{}, s)
}

type span struct {
	parent *span

	traceID     string
	spanID      string
	spanContext string

	startTime time.Time
	endTime   *time.Time

	err error

	inheritableAttributes map[string]interface{}
	attributes            map[string]interface{}
}

func (s *span) Duration() time.Duration {
	if s.endTime == nil {
		return 0
	}
	return s.endTime.Sub(s.startTime)
}

func (s *span) AddInheritableAttributes(attributes ...attribute) {
	if s.inheritableAttributes == nil {
		s.inheritableAttributes = make(map[string]interface{})
	}
	for _, attr := range attributes {
		s.inheritableAttributes[attr.key] = attr.value
	}
}

func (s *span) AddAttributes(attributes ...attribute) {
	if s.attributes == nil {
		s.attributes = make(map[string]interface{})
	}
	for _, attr := range attributes {
		s.attributes[attr.key] = attr.value
	}
}

func (s *span) recordError(err error) {
	s.err = err
}

func (s *span) end() {
	endTime := time.Now()
	s.endTime = &endTime
}

func Attribute(key string, value interface{}) attribute {
	return attribute{key: key, value: value}
}

type attribute struct {
	key   string
	value interface{}
}

// Key returns the attribute's key
func (a *attribute) Key() string {
	return a.key
}

// Value returns the attribute's value
func (a *attribute) Value() interface{} {
	return a.value
}
