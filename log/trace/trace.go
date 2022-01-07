package trace

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/theplant/appkit/log"
)

var _defaultIDGenerator = defaultIDGenerator()

func StartSpan(ctx context.Context, name string) (context.Context, *span) {
	var (
		parent  = fromContext(ctx)
		traceID TraceID
		spanID  = _defaultIDGenerator.NewSpanID()

		inheritableAttributes map[string]interface{}
	)

	if parent == nil {
		traceID = _defaultIDGenerator.NewTraceID()
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
		spanID:      spanID,
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
		dur      = s.Duration()
	)

	keysvals = append(keysvals,
		"ts", s.startTime.Format(time.RFC3339Nano),
		"trace.id", s.traceID,
		"span.id", s.spanID,
		"span.context", s.spanContext,
		"span.dur_ms", dur.Milliseconds(),
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
			"msg", fmt.Sprintf("%s (%v) -> %s (%T)", s.spanContext, dur, s.err, s.err),
			"span.err", s.err,
			"span.err_type", errType(s.err),
			"span.with_err", 1,
		)
		l.Error().Log(keysvals...)
	} else if r := recover(); r != nil {
		keysvals = append(keysvals,
			"msg", fmt.Sprintf("%s (%v) -> panic: %s (%T)", s.spanContext, dur, r, r),
			"span.err", r,
			"span.panic", 1,
			"span.err_type", errType(s.err),
			"span.with_err", 1,
		)
		l.Error().Log(keysvals...)
		panic(r)
	} else {
		keysvals = append(keysvals,
			"msg", fmt.Sprintf("%s (%v) -> success", s.spanContext, dur),
		)
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

	traceID     TraceID
	spanID      SpanID
	spanContext string

	startTime time.Time
	endTime   *time.Time

	err error

	inheritableAttributes map[string]interface{}
	attributes            map[string]interface{}

	mu sync.Mutex
}

func (s *span) Duration() time.Duration {
	if s.endTime == nil {
		return 0
	}
	return s.endTime.Sub(s.startTime)
}

func (s *span) AddInheritableAttributes(attributes ...attribute) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.inheritableAttributes == nil {
		s.inheritableAttributes = make(map[string]interface{})
	}
	for _, attr := range attributes {
		s.inheritableAttributes[attr.key] = attr.value
	}
}

func (s *span) AddAttributes(attributes ...attribute) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	if s.endTime != nil {
		return
	}

	endTime := time.Now()
	s.endTime = &endTime
}

// TraceID is a unique identity of a trace.
// nolint:revive // revive complains about stutter of `trace.TraceID`.
type TraceID [16]byte

var nilTraceID TraceID
var _ json.Marshaler = nilTraceID

// IsValid checks whether the trace TraceID is valid. A valid trace ID does
// not consist of zeros only.
func (t TraceID) IsValid() bool {
	return !bytes.Equal(t[:], nilTraceID[:])
}

// MarshalJSON implements a custom marshal function to encode TraceID
// as a hex string.
func (t TraceID) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// String returns the hex string representation form of a TraceID
func (t TraceID) String() string {
	return hex.EncodeToString(t[:])
}

// SpanID is a unique identity of a span in a trace.
type SpanID [8]byte

var nilSpanID SpanID
var _ json.Marshaler = nilSpanID

// IsValid checks whether the SpanID is valid. A valid SpanID does not consist
// of zeros only.
func (s SpanID) IsValid() bool {
	return !bytes.Equal(s[:], nilSpanID[:])
}

// MarshalJSON implements a custom marshal function to encode SpanID
// as a hex string.
func (s SpanID) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// String returns the hex string representation form of a SpanID
func (s SpanID) String() string {
	return hex.EncodeToString(s[:])
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
