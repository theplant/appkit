package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sts"
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

	cfg, err := aws.NewConfig(context.TODO(), logger, vault.Client, config.AWSPath)
	if err != nil {
		fmt.Println(err)
	}

	svc := sts.NewFromConfig(cfg)
	result, err := svc.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})

	fmt.Println(err, result)
}
