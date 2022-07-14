package relock

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"time"

	"github.com/gomodule/redigo/redis"
)

func getRandomString() string {
	b := make([]byte, 16)
	crand.Read(b)

	return base64.StdEncoding.EncodeToString(b)
}

func unlockInstance(ctx context.Context, client *redis.Pool, resource, value string) (bool, error) {
	conn := client.Get()
	defer conn.Close()

	// reply := conn.Eval(ctx, UnlockScript, []string{resource}, value)
	// if reply.Err() != nil {
	// 	return false, reply.Err()
	// }

	return true, nil
}

func lockInstance(ctx context.Context, client *redis.Pool, resource, value string, ttl time.Duration) (bool, error) {
	conn := client.Get()
	defer conn.Close()

	resultOperation, errSet := conn.Do("SetNX", resource, value)
	if errSet != nil {
		return false, errSet
	}

	if resultOperation.(int) == 0 {
		return false, ErrLockSingleRedis
	}

	return true, nil
}
