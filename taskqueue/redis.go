package taskqueue

import (
	"context"

	"github.com/go-redis/redis/v8"
)

type Redis interface {
	ScriptLoad(ctx context.Context, script string) *redis.StringCmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ZAdd(ctx context.Context, key string, members ...*redis.Z) *redis.IntCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}
