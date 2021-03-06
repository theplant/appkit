package vault

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/theplant/appkit/log"
)

type ServiceAccount struct {
	Client *api.Client
}

type Config struct {
	Address  string `default:"https://vault.vault.svc.cluster.local"`
	AuthPath string `default:"auth/kubernetes/login"`
	Role     string
	// *bool instead of bool to allow configuring the value to false
	Autorenew *bool `default:"true"`

	Token         string
	TokenFilename string `default:"/var/run/secrets/kubernetes.io/serviceaccount/token"`

	Disabled bool
}

type Client struct {
	*api.Client

	signal signal
}

// OnAuth calls f whenever the client authenticates. It's most useful
// for an auto-renewing client. If the client is already authenticated
// when OnAuth is called, f will be called immediately.
func (c *Client) OnAuth(f func() error) {
	// If we only call Wait, we might not be signalled in this case:
	//
	// 1. Create autorenewing client
	// 2. Client authenticates in separate goroutine
	// 3. Call OnAuth
	if c.Token() != "" {
		if f() != nil {
			return // Same behaviour as Wait()
		}
	}
	c.signal.wait(f)
}

func NewVaultClient(logger log.Logger, config Config) (*Client, error) {
	logger = logger.With(
		"context", "appkit/credentials/vault",
		"address", config.Address,
		"auth_path", config.AuthPath,
		"role", config.Role,
	)

	if config.Disabled {
		logger.Info().Log("msg", "not creating vault client: vault disabled by configuration")
		return nil, nil
	}

	cfg := api.Config{
		Address: config.Address,
	}

	token := config.Token
	if len(token) == 0 {
		tokBytes, err := ioutil.ReadFile(config.TokenFilename)

		if os.IsNotExist(err) {
			logger.Info().Log("msg", "not creating vault client: no token and no token file")
			return nil, nil
		}

		token = string(tokBytes)

		logger.Info().Log(
			"msg", fmt.Sprintf("creating vault client with token from %s", config.TokenFilename),
			"token_filename", config.TokenFilename,
		)
	} else {
		logger.Info().Log(
			"msg", "creating vault client with token from environment",
		)
	}

	client, err := api.NewClient(&cfg)
	if err != nil {
		return nil, errors.Wrap(err, "error in vault/api.NewClient")
	}

	c := &Client{Client: client, signal: newSignal()}
	if config.Autorenew == nil || *config.Autorenew {
		go autorenewAuthentication(c, token, config, logger)
	} else {
		_, err = fetchAuthToken(c, token, config, logger)
	}

	return c, err
}

func autorenewAuthentication(client *Client, token string, config Config, logger log.Logger) {
	logger = logger.With("autorenew", true)
	logger.Info().Log("msg", "starting automatic vault authentication renewal")

	sleeper := backoff.NewExponentialBackOff()
	sleeper.MaxInterval = 30 * time.Second

	for {
		var secret *api.Secret

		op := func() error {
			s, err := fetchAuthToken(client, token, config, logger)
			secret = s
			return err
		}

		notify := func(err error, next time.Duration) {
			logger.Error().Log(
				"msg", fmt.Sprintf("failed to authenticate with vault, will try again in %v: %v", next, err),
				"next_backoff", next,
				"err", err,
			)
		}

		err := backoff.RetryNotify(op, sleeper, notify)
		if err != nil {
			// There's no timeout on the sleeper, so if we get an error here, what can we do?
			logger.Crit().Log(
				"msg", fmt.Sprintf("error in backoff when authenticating with vault: %v", err),
				"err", err,
			)
		} else if secret == nil {
			logger.Crit().Log(
				"msg", "backoff returned no error, but we don't have a secret",
			)
		}

		l := logger.With("accessor", secret.Auth.Accessor)

		if secret.Auth.Renewable {

			// api.Client.NewRenewer only returns an error if
			// parameter is nil or parameter's secret is nil.
			renewer, _ := client.NewRenewer(&api.RenewerInput{
				Secret: secret,
			})

			go renewer.Renew()

		RenewalLoop:
			for {
				select {
				case err := <-renewer.DoneCh():
					if err != nil {
						l.WithError(errors.Wrap(err, "error renewing vault authentication")).Log()
					} else {
						l.Warn().Log(
							"msg", "halting vault authentication autorenewal",
						)
					}
					break RenewalLoop

				case renewal := <-renewer.RenewCh():
					l.Info().Log(
						"msg", fmt.Sprintf("renewed vault authentication at %s", renewal.RenewedAt),
						"renewed_at", renewal.RenewedAt,
					)
					LogWarnings(renewal.Secret, logger)
				}
			}
		}
	}
}

func fetchAuthToken(client *Client, token string, config Config, logger log.Logger) (*api.Secret, error) {
	logger.Debug().Log("msg", "authenticating with vault")
	authReq := map[string]interface{}{
		"jwt":  token,
		"role": config.Role,
	}

	s, err := client.Logical().Write(config.AuthPath, authReq)

	if err != nil {
		return s, errors.Wrap(err, "error authenticating with vault")
	}

	logger = logger.With("request_id", s.RequestID)

	if s.Auth == nil {
		logger.Warn().Log("msg", "vault auth path didn't return auth data")
	} else {
		expiry := time.Now().Add(time.Duration(s.Auth.LeaseDuration) * time.Second)
		logger.Info().Log(
			"msg", fmt.Sprintf("authenticated with vault, lease expires at %s", expiry),
			"accessor", s.Auth.Accessor,
			"policies", strings.Join(s.Auth.Policies, ","),
			"lease_duration", s.Auth.LeaseDuration,
			"renewable", s.Auth.Renewable,
		)
	}

	LogWarnings(s, logger)

	client.SetToken(s.Auth.ClientToken)

	client.signal.signal()

	return s, nil
}

func LogWarnings(s *api.Secret, logger log.Logger) {
	if len(s.Warnings) > 0 {
		l := logger.Warn()
		for _, w := range s.Warnings {
			l.Log("msg", fmt.Sprintf("vault api warning: %s", w))
		}
	}
}

////////////////////////////////////////

type signal struct {
	*sync.Cond
}

func newSignal() signal {
	var m sync.Mutex

	return signal{sync.NewCond(&m)}
}

func (s signal) signal() {
	s.Cond.Broadcast()
}

// Note that `f` will only be called the next time `signal` is called,
// it won't be called if `signal` was called before `wait`.
//
// 1. Create signal
// 2. Call Signal()
// 3. Call Wait(f)
// 4. Call Signal() => calls f
// 5. Call Wait(g)
// 6. Call Signal() => calls f, g
func (s signal) wait(f func() error) {
	go func() {
		for {
			s.Cond.L.Lock()
			s.Cond.Wait()
			s.Cond.L.Unlock()
			if f() != nil {
				break
			}
		}
	}()
}
