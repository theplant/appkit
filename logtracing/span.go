package logtracing

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

type span struct {
	parent *span

	traceID     TraceID
	spanID      SpanID
	spanContext string

	startTime time.Time
	endTime   time.Time

	err error

	keyvals []interface{}
	mu      sync.Mutex
}

func (s *span) IsRecording() bool {
	return s.endTime.IsZero()
}

func (s *span) RecordError(err error) {
	s.err = err
}

func (s *span) End() {
	if !s.IsRecording() {
		return
	}

	s.endTime = time.Now()
}

func (s *span) Duration() time.Duration {
	if s.IsRecording() {
		return 0
	}
	return s.endTime.Sub(s.startTime)
}

var ErrMissingValue = errors.New("(MISSING)")

func (s *span) AppendKVs(keyvals ...interface{}) {
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, ErrMissingValue)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.keyvals = append(s.keyvals, keyvals...)
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
