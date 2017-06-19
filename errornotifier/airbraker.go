// Package errornotifier is a notifier "provider" that provides
// a way to report runtime error. It uses gobrake notifier
// by default.
package errornotifier

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/theplant/appkit/log"
	gobrake "gopkg.in/airbrake/gobrake.v1"
)

// Notifier defines an interface for reporting error
type Notifier interface {
	Notify(interface{}, *http.Request) error
}

// airbrakeNotifier is a Notifier that sends messages to Airbrake,
// constructed via NewAirbrakeNotifier
type airbrakeNotifier struct {
	*gobrake.Notifier
}

// AirbrakeConfig is struct to embed into application config to allow
// configuration of Airbrake notifier from environment or other
// external source
type AirbrakeConfig struct {
	ProjectID   int64
	Token       string
	Environment string `default:"dev"`
}

// NewAirbrakeNotifier constructs Airbrake notifier from given config
//
// Returns error if no Airbrake configuration or airbrake
// configuration is invalid
func NewAirbrakeNotifier(c AirbrakeConfig) (Notifier, error) {
	if c.Token == "" {
		return nil, errors.New("blank Airbrake token")
	}

	if c.ProjectID <= 0 {
		return nil, fmt.Errorf("invalid Airbrake project id: %d", c.ProjectID)
	}

	airbrake := gobrake.NewNotifier(c.ProjectID, c.Token)
	airbrake.SetContext("environment", c.Environment)
	return airbrake, nil
}

type logNotifier struct {
	logger log.Logger
}

// Notify is part of Notifier interface
func (n *logNotifier) Notify(val interface{}, req *http.Request) error {
	logger := n.logger
	if req != nil {
		l, ok := log.FromContext(req.Context())
		if ok {
			logger = l
		}
	}

	return logger.Error().Log(
		"err", val,
		"msg", fmt.Sprintf("error notification: %v", val),
		"stack", fmt.Sprintf("%s", debug.Stack()),
	)
}

// NewLogNotifier constructs notifier that logs error notification
// messages to given logger
func NewLogNotifier(logger log.Logger) Notifier {
	return &logNotifier{
		logger: logger,
	}
}
