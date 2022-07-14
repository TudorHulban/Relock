package relock

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gomodule/redigo/redis"
)

const (
	// DefaultRetryCount is the max retry times for lock acquire
	DefaultRetryCount = 10

	// DefaultRetryDelay is upper wait time in millisecond for lock acquire retry
	DefaultRetryDelay = 200

	// ClockDriftFactor is clock drift factor, more information refers to doc
	ClockDriftFactor = 0.01

	// UnlockScript is redis lua script to release a lock
	UnlockScript = `
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("del", KEYS[1])
        else
            return 0
        end
        `
)

var (
	// ErrLockSingleRedis represents error when acquiring lock on a single redis
	ErrLockSingleRedis = errors.New("set lock on single redis failed")

	// ErrAcquireLock means acquire lock failed after max retry time
	ErrAcquireLock = errors.New("failed to require lock")
)

// RedLock holds the redis lock
type RedLock struct {
	clients []*redis.Pool
	cache   *MemoryCache

	driftFactor float64
	quorum      int
	RetryCount  uint
	RetryDelay  uint
}

// NewRedLock creates a RedLock
func NewRedLock(ctx context.Context, uris []string) (*RedLock, error) {
	var clients []*redis.Pool

	for _, uri := range uris {
		pool, errNew := NewRedisConnection(ctx, uri, DefaultConfigRedisData())
		if errNew != nil {
			return nil, errNew
		}

		clients = append(clients, pool)
	}

	return &RedLock{
		clients: clients,
		cache:   NewCache(),

		RetryCount:  DefaultRetryCount,
		RetryDelay:  DefaultRetryDelay,
		driftFactor: ClockDriftFactor,
		quorum:      len(uris)/2 + 1,
	}, nil
}

// Lock acquires a distribute lock, returns:
// a. the remaining valid duration that lock is guaranteed
// b. error if acquire lock fails
func (r *RedLock) Lock(ctx context.Context, resource string, ttl time.Duration) (time.Duration, error) {
	lockValue := getRandomString()

	for i := 0; i < int(r.RetryCount); i++ {
		start := time.Now()

		ctxCancel := int32(0)
		success := int32(0)

		cctx, cancel := context.WithTimeout(ctx, ttl)
		var wg sync.WaitGroup

		for _, client := range r.clients {
			cli := client
			wg.Add(1)

			go func() {
				defer wg.Done()

				locked, err := lockInstance(cctx, cli, resource, lockValue, ttl)
				if err == context.Canceled {
					atomic.AddInt32(&ctxCancel, 1)
				}

				if locked {
					atomic.AddInt32(&success, 1)
				}
			}()
		}

		wg.Wait()
		cancel()

		// fast fail, terminate acquiring lock if context is canceled
		if atomic.LoadInt32(&ctxCancel) > int32(0) {
			return 0, context.Canceled
		}

		drift := int(float64(ttl)*r.driftFactor) + 2
		costTime := time.Since(start).Nanoseconds()
		validityTime := int64(ttl) - costTime - int64(drift)

		if int(success) >= r.quorum && validityTime > 0 {
			r.cache.Set(resource, lockValue)

			return time.Duration(validityTime), nil
		}
		cctx, cancel = context.WithTimeout(ctx, ttl)

		for _, client := range r.clients {
			cli := client
			wg.Add(1)

			go func() {
				defer wg.Done()
				unlockInstance(cctx, cli, resource, lockValue) // nolint:errcheck
			}()
		}

		wg.Wait()
		cancel()

		// Wait a random delay before retrying
		time.Sleep(time.Duration(rand.Intn(int(r.RetryDelay))) * time.Millisecond)
	}

	return 0, ErrAcquireLock
}

// UnLock releases an acquired lock
func (r *RedLock) UnLock(ctx context.Context, lockID string) error {
	value, err := r.cache.Get(lockID)
	if err != nil {
		return err
	}

	if value == nil {
		return nil
	}

	defer r.cache.Delete(lockID)
	var wg sync.WaitGroup

	for _, client := range r.clients {
		cli := client
		wg.Add(1)

		go func() {
			defer wg.Done()

			unlockInstance(ctx, cli, lockID, value.(string))
		}()
	}

	wg.Wait()

	return nil
}
