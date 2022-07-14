package relock

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemoryCache(t *testing.T) {
	require := require.New(t)

	cache := NewCache()
	key := "1234"
	value := []byte("1")

	require.NoError(cache.Set(key, value))

	fetchedValue, errGet1 := cache.Get(key)
	t.Log(fetchedValue)
	t.Log(key)

	require.NoError(errGet1)
	require.Equal(value, fetchedValue, fmt.Sprintf("value: %#v\n", value))

	errDel := cache.Delete(key)
	require.NoError(errDel)

	shouldBeNil, errGet2 := cache.Get(key)
	require.Error(errGet2)
	require.Nil(shouldBeNil)
}
