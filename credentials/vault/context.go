package vault

import (
	"context"

	"github.com/hashicorp/vault/api"
)

type ctxKey int

const (
	key ctxKey = iota
)

func ForceContext(ctx context.Context) *api.Client {
	return ctx.Value(key).(*api.Client)
}

func Context(ctx context.Context, client *api.Client) context.Context {
	return context.WithValue(ctx, key, client)
}
