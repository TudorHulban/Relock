package relock

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _redisServers = []string{
	"tcp://127.0.0.1:6379",
	"tcp://127.0.0.1:6380",
	"tcp://127.0.0.1:6381",
}

func TestBasicLock(t *testing.T) {
	ctx := context.Background()
	lock, err := NewRedLock(ctx, _redisServers)
	require.Nil(t, err)

	_, err = lock.Lock(ctx, "foo", 200*time.Millisecond)
	assert.Nil(t, err)
	err = lock.UnLock(ctx, "foo")
	assert.Nil(t, err)
}

func TestUnlockExpiredKey(t *testing.T) {
	ctx := context.Background()
	lock, err := NewRedLock(ctx, _redisServers)
	assert.Nil(t, err)

	_, err = lock.Lock(ctx, "foo", 50*time.Millisecond)
	assert.Nil(t, err)
	time.Sleep(51 * time.Millisecond)
	err = lock.UnLock(ctx, "foo")
	assert.Nil(t, err)
}

const (
	fpath = "./counter.log"
)

func writer(count int, back chan *countResp) {
	ctx := context.Background()
	lock, err := NewRedLock(ctx, _redisServers)

	if err != nil {
		back <- &countResp{
			err: err,
		}
		return
	}

	incr := 0
	for i := 0; i < count; i++ {
		expiry, err := lock.Lock(ctx, "foo", 1000*time.Millisecond)
		if err != nil {
			log.Println(err)
		} else {
			if expiry > 500 {
				f, err := os.OpenFile(fpath, os.O_RDWR|os.O_CREATE, os.ModePerm)
				if err != nil {
					back <- &countResp{
						err: err,
					}
					return
				}

				buf := make([]byte, 1024)
				n, _ := f.Read(buf)
				num, _ := strconv.ParseInt(strings.TrimRight(string(buf[:n]), "\n"), 10, 64)
				f.WriteAt([]byte(strconv.Itoa(int(num+1))), 0)
				incr++

				f.Sync()
				f.Close()

				lock.UnLock(ctx, "foo")
			}
		}
	}
	back <- &countResp{
		count: incr,
		err:   nil,
	}
}

func init() {
	f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		panic(err)
	}
	f.WriteString("0")
	defer f.Close()
}

type countResp struct {
	count int
	err   error
}

func TestSimpleCounter(t *testing.T) {
	routines := 5
	inc := 100
	total := 0
	done := make(chan *countResp, routines)
	for i := 0; i < routines; i++ {
		go writer(inc, done)
	}
	for i := 0; i < routines; i++ {
		resp := <-done
		assert.Nil(t, resp.err)
		total += resp.count
	}

	f, err := os.OpenFile(fpath, os.O_RDONLY, os.ModePerm)
	assert.Nil(t, err)
	defer f.Close()
	buf := make([]byte, 1024)
	n, _ := f.Read(buf)
	counterInFile, _ := strconv.Atoi(string(buf[:n]))
	assert.Equal(t, total, counterInFile)
}

func TestNewRedLockError(t *testing.T) {
	ctx := context.Background()
	testCases := []struct {
		addrs   []string
		success bool
	}{
		{[]string{"127.0.0.1:6379"}, false},
		{[]string{"tcp://127.0.0.1:6379", "tcp://127.0.0.1:6380"}, false},
		{[]string{"tcp://127.0.0.1:6379", "tcp://127.0.0.1:6380", "tcp://127.0.0.1:6381"}, true},
	}
	for _, tc := range testCases {
		_, err := NewRedLock(ctx, tc.addrs)
		if tc.success {
			assert.Nil(t, err)
		} else {
			assert.NotNil(t, err)
		}
	}
}

func TestAcquireLockFailed(t *testing.T) {
	ctx := context.Background()
	servers := make([]string, 0, len(_redisServers))
	clis := make([]*redis.Client, 0, len(_redisServers))

	var wg sync.WaitGroup
	for idx, cli := range clis {
		// block two of redis instances
		if idx == 0 {
			continue
		}
		wg.Add(1)
		go func(c *redis.Client) {
			defer wg.Done()
			dur := 4 * time.Second
			c.ClientPause(ctx, dur)
			time.Sleep(dur)
		}(cli)
	}
	lock, err := NewRedLock(ctx, servers)
	assert.Nil(t, err)

	validity, err := lock.Lock(ctx, "foo", 100*time.Millisecond)
	assert.Equal(t, time.Duration(0), validity)
	assert.NotNil(t, err)

	wg.Wait()
}

func TestLockContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	clis := make([]*redis.Client, 0, len(_redisServers))
	var wg sync.WaitGroup

	for _, cli := range clis {
		wg.Add(1)
		go func(c *redis.Client) {
			defer wg.Done()
			c.ClientPause(ctx, time.Second)
			time.Sleep(time.Second)
		}(cli)
	}
	lock, err := NewRedLock(ctx, _redisServers)
	assert.Nil(t, err)

	cancel()
	_, err = lock.Lock(ctx, "foo", 100*time.Millisecond)
	assert.Equal(t, err, context.Canceled)
	wg.Wait()
}
