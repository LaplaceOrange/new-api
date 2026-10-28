package common

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetLocalChannelConcurrencyForTest() {
	channelConcurrencyLocalMu.Lock()
	channelConcurrencyLocal = make(map[string]map[string]time.Time)
	channelConcurrencyLocalMu.Unlock()
}

func TestChannelConcurrencyMemoryLimitLifecycle(t *testing.T) {
	previousRedisEnabled, previousRDB := RedisEnabled, RDB
	RedisEnabled = false
	RDB = nil
	resetLocalChannelConcurrencyForTest()
	t.Cleanup(func() {
		RedisEnabled, RDB = previousRedisEnabled, previousRDB
		resetLocalChannelConcurrencyForTest()
	})

	limit := 2
	first, acquired, err := AcquireChannelConcurrency(context.Background(), 1001, &limit)
	require.NoError(t, err)
	require.True(t, acquired)
	second, acquired, err := AcquireChannelConcurrency(context.Background(), 1001, &limit)
	require.NoError(t, err)
	require.True(t, acquired)
	third, acquired, err := AcquireChannelConcurrency(context.Background(), 1001, &limit)
	require.NoError(t, err)
	assert.False(t, acquired)
	assert.Nil(t, third)

	count, known, err := GetChannelConcurrency(context.Background(), 1001)
	require.NoError(t, err)
	assert.True(t, known)
	assert.Equal(t, 2, count)

	first.Release()
	first.Release()
	count, known, err = GetChannelConcurrency(context.Background(), 1001)
	require.NoError(t, err)
	assert.True(t, known)
	assert.Equal(t, 1, count)

	second.Release()
	count, _, err = GetChannelConcurrency(context.Background(), 1001)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestChannelConcurrencyMemoryUnlimitedAndExpiry(t *testing.T) {
	previousRedisEnabled, previousRDB := RedisEnabled, RDB
	RedisEnabled = false
	RDB = nil
	resetLocalChannelConcurrencyForTest()
	t.Cleanup(func() {
		RedisEnabled, RDB = previousRedisEnabled, previousRDB
		resetLocalChannelConcurrencyForTest()
	})

	for _, limit := range []*int{nil, func() *int { value := 0; return &value }()} {
		lease, acquired, err := AcquireChannelConcurrency(context.Background(), 1002, limit)
		require.NoError(t, err)
		require.True(t, acquired)
		lease.Release()
	}

	lease, acquired, err := AcquireChannelConcurrency(context.Background(), 1003, nil)
	require.NoError(t, err)
	require.True(t, acquired)
	channelConcurrencyLocalMu.Lock()
	channelConcurrencyLocal[lease.key][lease.token] = time.Now().Add(-time.Second)
	channelConcurrencyLocalMu.Unlock()
	count, known, err := GetChannelConcurrency(context.Background(), 1003)
	require.NoError(t, err)
	assert.True(t, known)
	assert.Equal(t, 0, count)
	lease.Release()
}

func TestSubscriptionConcurrencyMemoryLimitLifecycle(t *testing.T) {
	previousRedisEnabled, previousRDB := RedisEnabled, RDB
	RedisEnabled = false
	RDB = nil
	resetLocalChannelConcurrencyForTest()
	t.Cleanup(func() {
		RedisEnabled, RDB = previousRedisEnabled, previousRDB
		resetLocalChannelConcurrencyForTest()
	})

	first, acquired, err := AcquireSubscriptionConcurrency(1101, 1)
	require.NoError(t, err)
	require.True(t, acquired)
	second, acquired, err := AcquireSubscriptionConcurrency(1101, 1)
	require.NoError(t, err)
	assert.False(t, acquired)
	assert.Nil(t, second)

	count, known, err := GetSubscriptionConcurrency(context.Background(), 1101)
	require.NoError(t, err)
	assert.True(t, known)
	assert.Equal(t, 1, count)

	first.Release()
	count, _, err = GetSubscriptionConcurrency(context.Background(), 1101)
	require.NoError(t, err)
	assert.Zero(t, count)
}

func TestChannelAndModelConcurrencyLeaseLifecycle(t *testing.T) {
	previousRedisEnabled, previousRDB := RedisEnabled, RDB
	RedisEnabled = false
	RDB = nil
	resetLocalChannelConcurrencyForTest()
	t.Cleanup(func() {
		RedisEnabled, RDB = previousRedisEnabled, previousRDB
		resetLocalChannelConcurrencyForTest()
	})

	channelLimit, modelLimit := 2, 1
	first, acquired, err := AcquireChannelAndModelConcurrency(context.Background(), 1007, "gpt-4", &channelLimit, &modelLimit)
	require.NoError(t, err)
	require.True(t, acquired)
	second, acquired, err := AcquireChannelAndModelConcurrency(context.Background(), 1007, "gpt-4", &channelLimit, &modelLimit)
	require.NoError(t, err)
	assert.False(t, acquired)
	assert.Nil(t, second)
	channelCount, known, err := GetChannelConcurrency(context.Background(), 1007)
	require.NoError(t, err)
	assert.True(t, known)
	assert.Equal(t, 1, channelCount, "rejected model lease must release its channel lease")

	otherModel, acquired, err := AcquireChannelAndModelConcurrency(context.Background(), 1007, "gpt-4-mini", &channelLimit, &modelLimit)
	require.NoError(t, err)
	require.True(t, acquired, "a saturated model must not block another model")
	first.Release()
	otherModel.Release()
	channelCount, _, err = GetChannelConcurrency(context.Background(), 1007)
	require.NoError(t, err)
	assert.Equal(t, 0, channelCount)
}

func TestChannelConcurrencyMemoryConcurrentAcquireDoesNotOversell(t *testing.T) {
	previousRedisEnabled, previousRDB := RedisEnabled, RDB
	RedisEnabled = false
	RDB = nil
	resetLocalChannelConcurrencyForTest()
	t.Cleanup(func() {
		RedisEnabled, RDB = previousRedisEnabled, previousRDB
		resetLocalChannelConcurrencyForTest()
	})

	limit := 3
	var mu sync.Mutex
	leases := make([]*ChannelConcurrencyLease, 0, 10)
	acquiredCount := 0
	var acquireErrors []error
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lease, acquired, err := AcquireChannelConcurrency(context.Background(), 1004, &limit)
			if err != nil {
				mu.Lock()
				acquireErrors = append(acquireErrors, err)
				mu.Unlock()
				return
			}
			if !acquired {
				return
			}
			mu.Lock()
			acquiredCount++
			leases = append(leases, lease)
			mu.Unlock()
		}()
	}
	wg.Wait()
	assert.Empty(t, acquireErrors)
	assert.Equal(t, 3, acquiredCount)
	count, known, err := GetChannelConcurrency(context.Background(), 1004)
	require.NoError(t, err)
	assert.True(t, known)
	assert.Equal(t, 3, count)
	for _, lease := range leases {
		lease.Release()
	}
}

func TestChannelConcurrencyRedisAtomicLifecycleAndFailureOpen(t *testing.T) {
	previousRedisEnabled, previousRDB := RedisEnabled, RDB
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	require.NoError(t, client.Ping(context.Background()).Err())
	RedisEnabled = true
	RDB = client
	resetLocalChannelConcurrencyForTest()
	t.Cleanup(func() {
		_ = client.Close()
		RedisEnabled, RDB = previousRedisEnabled, previousRDB
		resetLocalChannelConcurrencyForTest()
	})

	limit := 2
	first, acquired, err := AcquireChannelConcurrency(context.Background(), 1005, &limit)
	require.NoError(t, err)
	require.True(t, acquired)
	second, acquired, err := AcquireChannelConcurrency(context.Background(), 1005, &limit)
	require.NoError(t, err)
	require.True(t, acquired)
	third, acquired, err := AcquireChannelConcurrency(context.Background(), 1005, &limit)
	require.NoError(t, err)
	assert.False(t, acquired)
	assert.Nil(t, third)

	count, known, err := GetChannelConcurrency(context.Background(), 1005)
	require.NoError(t, err)
	assert.True(t, known)
	assert.Equal(t, 2, count)

	server.FastForward(channelConcurrencyLeaseTTL + time.Second)
	count, known, err = GetChannelConcurrency(context.Background(), 1005)
	require.NoError(t, err)
	assert.True(t, known)
	assert.Equal(t, 0, count)

	first.Release()
	second.Release()
	modelLimit := 1
	combined, acquired, err := AcquireChannelAndModelConcurrency(context.Background(), 1008, "model", &limit, &modelLimit)
	require.NoError(t, err)
	require.True(t, acquired)
	denied, acquired, err := AcquireChannelAndModelConcurrency(context.Background(), 1008, "model", &limit, &modelLimit)
	require.NoError(t, err)
	assert.False(t, acquired)
	assert.Nil(t, denied)
	count, _, err = GetChannelConcurrency(context.Background(), 1008)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "failed model admission rolls back its Redis channel slot")
	combined.Release()
	combined.Release()
	count, _, err = GetChannelConcurrency(context.Background(), 1008)
	require.NoError(t, err)
	assert.Zero(t, count)
	server.Close()
	lease, acquired, err := AcquireChannelConcurrency(context.Background(), 1006, &limit)
	require.NoError(t, err)
	assert.True(t, acquired)
	assert.Nil(t, lease)
	_, known, err = GetChannelConcurrency(context.Background(), 1006)
	assert.False(t, known)
	assert.Error(t, err)
}
