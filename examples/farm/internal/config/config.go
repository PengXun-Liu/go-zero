package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
)

// Config holds the full application configuration.
type Config struct {
	rest.RestConf

	// Auth holds JWT configuration.
	Auth struct {
		AccessSecret string
		AccessExpire int64
	}

	// Redis holds the Redis connection configuration used for game state storage.
	Redis redis.RedisConf
}
