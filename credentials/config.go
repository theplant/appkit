package credentials

import (
	"fmt"

	"github.com/theplant/appkit/credentials/vault"
)

type Config struct {
	Authn   vault.Config
	AWSPath string
}

func WithServiceName(cfg Config, name string) Config {
	if cfg.AWSPath == "" {
		cfg.AWSPath = fmt.Sprintf("aws/sts/%s", name)
	}

	if cfg.Authn.Role == "" {
		cfg.Authn.Role = name
	}

	// Non-pointer, so received a copy. Lets return the copy.
	return cfg
}
