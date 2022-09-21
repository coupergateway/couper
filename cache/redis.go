package cache

import (
	"context"

	"github.com/go-redis/redis/v8"
)

type RedisStore struct {
	client *redis.Client
	ctx    context.Context
}

// func NewRedis(ctx context.Context, configURL string) (Storage, error) {
// 	opts, err := redis.ParseURL(configURL)
// 	if err != nil {
// 		return nil, err
// 	}

// 	client := redis.NewClient(opts)

// 	return &RedisStore{
// 		client: client,
// 		ctx:    ctx,
// 	}, nil
// }

// func (r *RedisStore) Del(key string) {
// 	r.client.Del(r.ctx, key)
// }

// func (r *RedisStore) Get(key string) (interface{}, error) {
// 	return r.client.Get(r.ctx, key).Result()
// }

// func (r *RedisStore) Set(key string, val interface{}, ttl int64) {
// 	r.client.Set(r.ctx, key, val, time.Duration(ttl*int64(time.Second)))
// }
