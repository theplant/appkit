package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/theplant/appkit/credentials/aws"
	"github.com/theplant/appkit/credentials/vault"
	"github.com/theplant/appkit/log"
)

func main() {
	logger := log.Default()

	logger.Info().Log("msg", "starting up...")

	vault, err := vault.NewVaultClient(logger, vault.Config{
		Address:  "http://vault:8200",
		AuthPath: "auth/kubernetes/login",
		Role:     os.Getenv("VAULT_AUTHN_ROLE"),
		//		Autorenew: true,
	})

	if err != nil {
		fmt.Println(err)
	}

	session, err := aws.NewSession(logger, vault, os.Getenv("VAULT_AWS_PATH"))
	if err != nil {
		fmt.Println(err)
	}

	svc := sts.New(session)
	result, err := svc.GetCallerIdentity(nil)

	fmt.Println(err, result)
}
