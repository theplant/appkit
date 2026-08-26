package monitoring

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type recordingMonitor struct {
	Monitor
	measurements chan string
}

func (m *recordingMonitor) InsertRecord(measurement string, value interface{}, tags map[string]string, fields map[string]interface{}, t time.Time) {
	m.measurements <- measurement
}

func serve(t *testing.T, mw func(http.Handler) http.Handler) (monitorInContext bool) {
	t.Helper()

	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, monitorInContext = r.Context().Value(monitorKey).(Monitor)
		w.WriteHeader(http.StatusOK)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/orders/123", nil))

	return
}

func TestWithMonitorRecordsRequestMetric(t *testing.T) {
	m := &recordingMonitor{measurements: make(chan string, 1)}

	if !serve(t, WithMonitor(m)) {
		t.Error("expected the monitor to be installed in the request context")
	}

	// InsertRecord runs in a goroutine started by a deferred func, so it may
	// not have run by the time ServeHTTP returns.
	select {
	case measurement := <-m.measurements:
		if measurement != "request" {
			t.Errorf(`expected measurement "request", got %q`, measurement)
		}
	case <-time.After(time.Second):
		t.Error("expected a request metric, got none")
	}
}

func TestWithMonitorContextOnlySkipsRequestMetric(t *testing.T) {
	m := &recordingMonitor{measurements: make(chan string, 1)}

	if !serve(t, WithMonitorContextOnly(m)) {
		t.Error("expected the monitor to be installed in the request context")
	}

	select {
	case measurement := <-m.measurements:
		t.Errorf("expected no request metric, got %q", measurement)
	case <-time.After(100 * time.Millisecond):
	}
}
