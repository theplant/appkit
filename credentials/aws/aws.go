package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/theplant/appkit/credentials/vault"
	"github.com/theplant/appkit/kerrs"
	"github.com/theplant/appkit/log"
)

type vaultProvider struct {
	credentials.Expiry

	vault *api.Client

	path string

	logger log.Logger
}

// Retrieve is half of the `github.com/aws/aws/credentials.Provider`
// interface.
//
// IsExpired (the other half) is implemented via credentials.Expiry
// and `SetExpiration`.
func (v *vaultProvider) Retrieve() (credentials.Value, error) {
	v.logger.Debug().Log("msg", "renewing aws credentials")

	s, err := v.vault.Logical().Read(v.path)

	if err != nil {
		return credentials.Value{}, errors.Wrap(err, "error renewing aws credentials via vault")
	}

	l := v.logger.With("request_id", s.RequestID)

	vault.LogWarnings(s, l)

	expiry, err := s.TokenTTL()
	if err != nil {
		return credentials.Value{}, errors.Wrap(err, "error calculating credentials ttl")
	}

	v.SetExpiration(time.Now().Add(expiry), 0)

	accessKey, err := validate(s.Data, "access_key", nil)
	secretKey, err := validate(s.Data, "secret_key", err)
	sessionToken, err := validate(s.Data, "security_token", err)

	if err != nil {
		return credentials.Value{}, errors.Wrap(err, "vault data doesn't contain valid credentials")
	}

	l.Info().Log(
		"msg", fmt.Sprintf("renewed aws credentials, lease expires at %s", time.Now().Add(expiry)),
		"lease_id", s.LeaseID,
		"lease_duration", s.LeaseDuration,
		"renewable", s.Renewable,
	)

	return credentials.Value{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		SessionToken:    sessionToken,
		ProviderName:    "VaultProvider",
	}, nil
}

func validate(m map[string]interface{}, key string, err error) (string, error) {
	s, ok := m[key].(string)
	if !ok {
		return "", kerrs.Append(err, fmt.Errorf("%q missing in auth data", key))
	}
	return s, nil
}

func NewSession(logger log.Logger, vault *api.Client, path string) (*session.Session, error) {
	logger = logger.With(
		"context", "appkit/credentials/aws",
	)

	if vault != nil {
		logger.Info().Log(
			"msg", "initialising aws session with vault",
			"path", path,
		)

		config := aws.NewConfig().WithCredentials(
			credentials.NewCredentials(
				&vaultProvider{
					vault:  vault,
					path:   path,
					logger: logger,
				}))

		s, err := session.NewSessionWithOptions(session.Options{
			Config: *config,
		})

		if err != nil {
			return nil, errors.Wrap(err, "error creating aws session")
		}

		return s, nil
	}

	logger.Info().Log("msg", "using default aws session configuration")

	return session.NewSession()
}
