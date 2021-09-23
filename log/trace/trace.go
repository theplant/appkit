package trace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/theplant/appkit/log"
)

func StartSpan(ctx context.Context, name string) (context.Context, *span) {
	var (
		parent = fromContext(ctx)

		traceID               string
		parentSpanID          string
		inheritableAttributes map[string]interface{}
	)

	if parent == nil {
		traceID = uuid.New().String()
	} else {
		traceID = parent.traceID
		parentSpanID = parent.spanID
		if len(parent.inheritableAttributes) > 0 {
			inheritableAttributes = make(map[string]interface{})
			for k, v := range parent.inheritableAttributes {
				inheritableAttributes[k] = v
			}
		}
	}

	s := span{
		traceID:      traceID,
		spanParentID: parentSpanID,
		spanID:       uuid.New().String(),
		spanContext:  name,

		startTime: time.Now(),

		inheritableAttributes: inheritableAttributes,
	}

	return newContext(ctx, &s), &s
}

func EndSpan(l log.Logger, s *span, err error) {
	s.recordError(err)
	s.end()
	logSpan(l, s)
}

func logSpan(log log.Logger, s *span) {
	dur := s.endTime.Sub(s.startTime).Milliseconds()
	l := log.With(
		"ts", s.startTime.Format(time.RFC3339Nano),
		"trace.id", s.traceID,
		"span.id", s.spanID,
		"span.context", s.spanContext,
		"span.dur_ms", dur,
	)

	if s.spanParentID != "" {
		l = l.With("span.parent_id", s.spanParentID)
	}

	var keyvals []interface{}
	for k, v := range s.inheritableAttributes {
		if _, ok := s.attributes[k]; ok {
			continue
		}
		keyvals = append(keyvals, k, v)
	}
	for k, v := range s.attributes {
		keyvals = append(keyvals, k, v)
	}
	l = l.With(keyvals...)

	if s.err != nil {
		l.Error().Log(
			"msg", fmt.Sprintf("%s (%v) -> %s (%T)", s.spanContext, dur, s.err, s.err),
			"span.err", s.err,
			"span.err_type", errType(s.err),
			"span.with_err", 1,
		)
	} else if r := recover(); r != nil {
		l.Error().Log(
			"msg", fmt.Sprintf("%s (%v) -> panic: %s (%T)", s.spanContext, dur, r, r),
			"span.err", r,
			"span.panic", 1,
			"span.err_type", errType(s.err),
			"span.with_err", 1,
		)
		panic(r)
	} else {
		l.Info().Log("msg", fmt.Sprintf("%s (%v) -> success", s.spanContext, dur))
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
	traceID      string
	spanParentID string
	spanID       string
	spanContext  string

	startTime time.Time
	endTime   time.Time

	err error

	inheritableAttributes map[string]interface{}
	attributes            map[string]interface{}
}

func (s *span) AddInheritableAttributes(attributes ...Attribute) {
	if s.inheritableAttributes == nil {
		s.inheritableAttributes = make(map[string]interface{})
	}
	for _, attr := range attributes {
		s.inheritableAttributes[attr.key] = attr.value
	}
}

func (s *span) AddAttributes(attributes ...Attribute) {
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
	s.endTime = time.Now()
}

// Construct with one of: BoolAttribute, Int64Attribute, or StringAttribute.
type Attribute struct {
	key   string
	value interface{}
}

// Key returns the attribute's key
func (a *Attribute) Key() string {
	return a.key
}

// Value returns the attribute's value
func (a *Attribute) Value() interface{} {
	return a.value
}

// BoolAttribute returns a bool-valued attribute.
func BoolAttribute(key string, value bool) Attribute {
	return Attribute{key: key, value: value}
}

// Int64Attribute returns an int64-valued attribute.
func Int64Attribute(key string, value int64) Attribute {
	return Attribute{key: key, value: value}
}

// Float64Attribute returns a float64-valued attribute.
func Float64Attribute(key string, value float64) Attribute {
	return Attribute{key: key, value: value}
}

// StringAttribute returns a string-valued attribute.
func StringAttribute(key string, value string) Attribute {
	return Attribute{key: key, value: value}
}
