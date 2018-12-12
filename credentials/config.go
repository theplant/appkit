package credentials

import (
	"github.com/theplant/appkit/credentials/vault"
)

type Config struct {
	Authn   vault.Config
	AWSPath string
}
