package relock

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func isSocketOpened(socket string) bool {
	conn, err := net.DialTimeout("tcp", socket, time.Second)
	if err != nil {
		return false
	}

	defer conn.Close()

	return conn != nil
}

func TestRedisConnection(t *testing.T) {
	sock := "localhost:6379"
	ctx := context.Background()

	pool, errNew := NewRedisConnection(ctx, sock, DefaultConfigRedisData())
	if isSocketOpened(sock) {
		require.NoError(t, errNew)
		require.NotZero(t, pool)

		return
	}

	require.Error(t, errNew)
	require.Nil(t, pool)
}
