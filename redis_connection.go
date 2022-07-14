package relock

import (
	"context"
	"time"

	"github.com/gomodule/redigo/redis"
)

type ConfigRedisData struct {
	maxIdlePoolConnections   uint
	maxActivePoolConnections uint
	databaseNumber           uint
}

func NewConfigRedisData() (*ConfigRedisData, error) {
	return nil, nil
}

func (d ConfigRedisData) validate() error {
	return nil
}

func DefaultConfigRedisData() *ConfigRedisData {
	return &ConfigRedisData{
		maxIdlePoolConnections:   80,
		maxActivePoolConnections: 12000,
		databaseNumber:           1,
	}
}

func NewRedisConnection(ctx context.Context, sock string, data *ConfigRedisData) (*redis.Pool, error) {
	var errConn error

	pool := redis.Pool{
		MaxIdle:   int(data.maxIdlePoolConnections),
		MaxActive: int(data.maxActivePoolConnections),
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialContext(ctx, "tcp", sock, redis.DialDatabase(int(data.databaseNumber)))
			if err != nil {
				errConn = err
			}

			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}

	if errConn != nil {
		return nil, errConn
	}

	return &pool, nil
}
