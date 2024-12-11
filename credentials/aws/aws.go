package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/theplant/appkit/credentials/vault"
	"github.com/theplant/appkit/kerrs"
	"github.com/theplant/appkit/log"
)

// vaultProvider implements `github.com/aws/aws-sdk-go-v2/aws.CredentialsProvider`
type vaultProvider struct {
	vault *api.Client

	path string

	logger log.Logger
}

func (v *vaultProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	v.logger.Debug().Log("msg", "renewing aws credentials")

	s, err := v.vault.Logical().Read(v.path)

	if err != nil {
		return aws.Credentials{}, errors.Wrap(err, "error renewing aws credentials via vault")
	}

	l := v.logger.With("request_id", s.RequestID)

	vault.LogWarnings(s, l)

	expiry, err := s.TokenTTL()
	if err != nil {
		return aws.Credentials{}, errors.Wrap(err, "error calculating credentials ttl")
	}
	expireAt := time.Now().Add(expiry)

	accessKey, err := validate(s.Data, "access_key", nil)
	secretKey, err := validate(s.Data, "secret_key", err)
	sessionToken, err := validate(s.Data, "security_token", err)

	if err != nil {
		return aws.Credentials{}, errors.Wrap(err, "vault data doesn't contain valid credentials")
	}

	l.Info().Log(
		"msg", fmt.Sprintf("renewed aws credentials, lease expires at %s", expireAt),
		"lease_id", s.LeaseID,
		"lease_duration", s.LeaseDuration,
		"renewable", s.Renewable,
	)

	return aws.Credentials{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		SessionToken:    sessionToken,
		Source:          "VaultProvider",
		CanExpire:       true,
		Expires:         expireAt,
	}, nil
}

func validate(m map[string]interface{}, key string, err error) (string, error) {
	s, ok := m[key].(string)
	if !ok {
		return "", kerrs.Append(err, fmt.Errorf("%q missing in auth data", key))
	}
	return s, nil
}

func NewConfig(ctx context.Context, logger log.Logger, vault *api.Client, path string) (aws.Config, error) {
	logger = logger.With(
		"context", "appkit/credentials/aws",
		"aws_secret_path", path,
	)

	var opts []func(*config.LoadOptions) error

	if vault != nil {
		logger.Info().Log(
			"msg", "with vault-backed credentials provider",
		)

		opts = append(opts, config.WithCredentialsProvider(&vaultProvider{
			vault:  vault,
			path:   path,
			logger: logger,
		}))
	}

	logger.Info().Log("msg", "loading aws config")

	return config.LoadDefaultConfig(ctx, opts...)
}
