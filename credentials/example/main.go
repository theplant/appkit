package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/jinzhu/configor"
	"github.com/theplant/appkit/credentials"
	"github.com/theplant/appkit/credentials/aws"
	"github.com/theplant/appkit/credentials/vault"
	"github.com/theplant/appkit/log"
)

func main() {
	logger := log.Default()

	logger.Info().Log("msg", "starting up...")

	var config credentials.Config

	err := configor.New(&configor.Config{ENVPrefix: "VAULT"}).Load(&config)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("%#v (autorenew: %v)\n", config, *config.Authn.Autorenew)

	vault, err := vault.NewVaultClient(logger, config.Authn)

	if err != nil {
		fmt.Println(err)
	}

	session, err := aws.NewSession(logger, vault.Client, config.AWSPath)
	if err != nil {
		fmt.Println(err)
	}

	svc := sts.New(session)
	result, err := svc.GetCallerIdentity(nil)

	fmt.Println(err, result)
}
