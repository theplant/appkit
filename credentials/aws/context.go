package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/session"
)

type key int

const (
	awsKey key = iota
)

func ForceContext(ctx context.Context) *session.Session {
	return ctx.Value(awsKey).(*session.Session)
}

func Context(ctx context.Context, s *session.Session) context.Context {
	return context.WithValue(ctx, awsKey, s)
}
