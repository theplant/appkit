package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type key int

const (
	awsKey key = iota
)

func ForceContext(ctx context.Context) aws.Config {
	return ctx.Value(awsKey).(aws.Config)
}

func Context(ctx context.Context, cfg aws.Config) context.Context {
	return context.WithValue(ctx, awsKey, cfg)
}
