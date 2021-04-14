// Package errornotifier is a notifier "provider" that provides
// a way to report runtime error. It uses gobrake notifier
// by default.
package errornotifier

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"

	"github.com/airbrake/gobrake"
	"github.com/theplant/appkit/log"
)

// Notifier defines an interface for reporting error
type Notifier interface {
	Notify(err interface{}, r *http.Request, context map[string]interface{})
}

// airbrakeNotifier is a Notifier that sends messages to Airbrake,
// constructed via NewAirbrakeNotifier
type airbrakeNotifier struct {
	notifier *gobrake.Notifier
}

func (n *airbrakeNotifier) Notify(e interface{}, req *http.Request, context map[string]interface{}) {
	notice := n.notifier.Notice(e, req, 1)
	for k, v := range context {
		notice.Context[k] = v
	}
	n.notifier.SendNoticeAsync(notice)
}

// AirbrakeConfig is struct to embed into application config to allow
// configuration of Airbrake notifier from environment or other
// external source
type AirbrakeConfig struct {
	ProjectID   int64
	Token       string
	Environment string `default:"dev"`

	KeysBlocklist []interface{}
}

var defaultKeysBlocklist = []interface{}{
	"Authorization",
}

// NewAirbrakeNotifier constructs Airbrake notifier from given config
//
// Returns error if no Airbrake configuration or airbrake
// configuration is invalid
//
// Notify is async, call close to wait send data to Airbrake.
func NewAirbrakeNotifier(c AirbrakeConfig) (Notifier, io.Closer, error) {
	if c.Token == "" {
		return nil, nil, errors.New("blank Airbrake token")
	}

	if c.ProjectID <= 0 {
		return nil, nil, fmt.Errorf("invalid Airbrake project id: %d", c.ProjectID)
	}

	if c.KeysBlocklist == nil {
		c.KeysBlocklist = defaultKeysBlocklist
	}

	notifier := gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		ProjectId:     c.ProjectID,
		ProjectKey:    c.Token,
		Environment:   c.Environment,
		KeysBlacklist: c.KeysBlocklist,
	})

	return &airbrakeNotifier{notifier: notifier}, notifier, nil
}

type logNotifier struct {
	logger log.Logger
}

// Notify is part of Notifier interface
func (n *logNotifier) Notify(val interface{}, req *http.Request, context map[string]interface{}) {
	logger := n.logger
	if req != nil {
		l, ok := log.FromContext(req.Context())
		if ok {
			logger = l
		}
	}

	_ = logger.Error().Log(
		"err", val,
		"context", fmt.Sprint(context),
		"msg", fmt.Sprintf("error notification: %v", val),
		"stacktrace", string(debug.Stack()),
	)
}

// NewLogNotifier constructs notifier that logs error notification
// messages to given logger
func NewLogNotifier(logger log.Logger) Notifier {
	return &logNotifier{
		logger: logger,
	}
}
