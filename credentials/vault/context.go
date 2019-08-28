package vault

import (
	"context"
)

type ctxKey int

const (
	key ctxKey = iota
)

func ForceContext(ctx context.Context) *Client {
	return ctx.Value(key).(*Client)
}

func Context(ctx context.Context, client *Client) context.Context {
	return context.WithValue(ctx, key, client)
}
