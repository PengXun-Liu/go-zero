package svc

import (
	"github.com/zeromicro/go-zero/examples/farm/internal/config"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// ServiceContext holds shared dependencies available to all handlers and logic layers.
type ServiceContext struct {
	Config config.Config
	Redis  *redis.Redis
}

// NewServiceContext creates a new ServiceContext with the given config.
func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		Redis:  redis.MustNewRedis(c.Redis),
	}
}
