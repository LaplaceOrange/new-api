package common

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	channelConcurrencyKeyPrefix = "new-api:channel-concurrency:v1:"
	channelConcurrencyLeaseTTL  = 2 * time.Minute
)

var (
	channelConcurrencyLocalMu sync.Mutex
	channelConcurrencyLocal   = make(map[int]map[string]time.Time)
)

// ChannelConcurrencyLease represents one formal relay request occupying a
// channel slot. Release is idempotent and should be deferred by the caller.
type ChannelConcurrencyLease struct {
	channelID int
	token     string
	redis     bool
	cancel    context.CancelFunc
	once      sync.Once
}

func (lease *ChannelConcurrencyLease) ChannelID() int {
	if lease == nil {
		return 0
	}
	return lease.channelID
}

// AcquireChannelConcurrency atomically reserves one slot for a channel.
// A nil or non-positive limit means unlimited. Redis failures fail open and
// return an untracked lease so an outage does not block all relay traffic.
func AcquireChannelConcurrency(ctx context.Context, channelID int, limit *int) (*ChannelConcurrencyLease, bool, error) {
	if channelID <= 0 {
		return nil, true, nil
	}
	max := 0
	if limit != nil && *limit > 0 {
		max = *limit
	}
	token := uuid.NewString()
	if RedisEnabled && RDB != nil {
		lease, acquired, err := acquireRedisChannelConcurrency(ctx, channelID, token, max)
		if err == nil {
			return lease, acquired, nil
		}
		SysError(fmt.Sprintf("channel concurrency Redis acquire failed: channel_id=%d err=%v; allowing request", channelID, err))
		return nil, true, nil
	}

	channelConcurrencyLocalMu.Lock()
	defer channelConcurrencyLocalMu.Unlock()
	entries := localChannelConcurrencyEntries(channelID)
	now := time.Now()
	for leaseToken, expiresAt := range entries {
		if !expiresAt.After(now) {
			delete(entries, leaseToken)
		}
	}
	if max > 0 && len(entries) >= max {
		return nil, false, nil
	}
	entries[token] = now.Add(channelConcurrencyLeaseTTL)
	return newChannelConcurrencyLease(channelID, token, false, ctx), true, nil
}

// GetChannelConcurrency returns the current number of formal relay leases.
// known is false when Redis is unavailable, because fail-open requests cannot
// be represented accurately in a shared counter.
func GetChannelConcurrency(ctx context.Context, channelID int) (count int, known bool, err error) {
	if channelID <= 0 {
		return 0, true, nil
	}
	if RedisEnabled && RDB != nil {
		count, err = countRedisChannelConcurrency(ctx, channelID)
		if err != nil {
			return 0, false, err
		}
		return count, true, nil
	}

	channelConcurrencyLocalMu.Lock()
	defer channelConcurrencyLocalMu.Unlock()
	entries := localChannelConcurrencyEntries(channelID)
	now := time.Now()
	for leaseToken, expiresAt := range entries {
		if !expiresAt.After(now) {
			delete(entries, leaseToken)
		}
	}
	return len(entries), true, nil
}

func GetChannelsConcurrency(ctx context.Context, channelIDs []int) (map[int]int, map[int]bool) {
	counts := make(map[int]int, len(channelIDs))
	known := make(map[int]bool, len(channelIDs))
	for _, channelID := range channelIDs {
		count, isKnown, err := GetChannelConcurrency(ctx, channelID)
		if err != nil {
			SysError(fmt.Sprintf("channel concurrency read failed: channel_id=%d err=%v", channelID, err))
			known[channelID] = false
			continue
		}
		counts[channelID] = count
		known[channelID] = isKnown
	}
	return counts, known
}

func (lease *ChannelConcurrencyLease) Release() {
	if lease == nil || lease.token == "" {
		return
	}
	lease.once.Do(func() {
		if lease.cancel != nil {
			lease.cancel()
		}
		if lease.redis {
			if err := releaseRedisChannelConcurrency(lease.channelID, lease.token); err != nil {
				SysError(fmt.Sprintf("channel concurrency Redis release failed: channel_id=%d err=%v", lease.channelID, err))
			}
			return
		}
		channelConcurrencyLocalMu.Lock()
		if entries, ok := channelConcurrencyLocal[lease.channelID]; ok {
			delete(entries, lease.token)
			if len(entries) == 0 {
				delete(channelConcurrencyLocal, lease.channelID)
			}
		}
		channelConcurrencyLocalMu.Unlock()
	})
}

func newChannelConcurrencyLease(channelID int, token string, useRedis bool, parent context.Context) *ChannelConcurrencyLease {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	lease := &ChannelConcurrencyLease{channelID: channelID, token: token, redis: useRedis, cancel: cancel}
	go renewChannelConcurrencyLease(ctx, lease)
	return lease
}

func renewChannelConcurrencyLease(ctx context.Context, lease *ChannelConcurrencyLease) {
	ticker := time.NewTicker(channelConcurrencyLeaseTTL / 3)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if lease.redis {
				if err := renewRedisChannelConcurrency(lease.channelID, lease.token); err != nil {
					SysError(fmt.Sprintf("channel concurrency Redis renew failed: channel_id=%d err=%v", lease.channelID, err))
				}
				continue
			}
			channelConcurrencyLocalMu.Lock()
			if entries, ok := channelConcurrencyLocal[lease.channelID]; ok {
				if _, exists := entries[lease.token]; exists {
					entries[lease.token] = time.Now().Add(channelConcurrencyLeaseTTL)
				}
			}
			channelConcurrencyLocalMu.Unlock()
		}
	}
}

func localChannelConcurrencyEntries(channelID int) map[string]time.Time {
	entries := channelConcurrencyLocal[channelID]
	if entries == nil {
		entries = make(map[string]time.Time)
		channelConcurrencyLocal[channelID] = entries
	}
	return entries
}

const acquireChannelConcurrencyScript = `
local now = tonumber(ARGV[1])
local ttl = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now)
local count = redis.call('ZCARD', KEYS[1])
if limit > 0 and count >= limit then
  return {0, count}
end
redis.call('ZADD', KEYS[1], now + ttl, ARGV[4])
redis.call('PEXPIRE', KEYS[1], ttl)
return {1, count + 1}
`

const renewChannelConcurrencyScript = `
local now = tonumber(ARGV[1])
local ttl = tonumber(ARGV[2])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now)
if redis.call('ZSCORE', KEYS[1], ARGV[3]) == false then
  return 0
end
redis.call('ZADD', KEYS[1], now + ttl, ARGV[3])
redis.call('PEXPIRE', KEYS[1], ttl)
return 1
`

const releaseChannelConcurrencyScript = `
redis.call('ZREM', KEYS[1], ARGV[1])
if redis.call('ZCARD', KEYS[1]) == 0 then
  redis.call('DEL', KEYS[1])
end
return 1
`

const countChannelConcurrencyScript = `
local now = tonumber(ARGV[1])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now)
return redis.call('ZCARD', KEYS[1])
`

func channelConcurrencyKey(channelID int) string {
	return fmt.Sprintf("%s%d", channelConcurrencyKeyPrefix, channelID)
}

func redisNowMilliseconds() int64 {
	return time.Now().UnixMilli()
}

func acquireRedisChannelConcurrency(ctx context.Context, channelID int, token string, limit int) (*ChannelConcurrencyLease, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := RDB.Eval(ctx, acquireChannelConcurrencyScript, []string{channelConcurrencyKey(channelID)}, redisNowMilliseconds(), channelConcurrencyLeaseTTL.Milliseconds(), limit, token).Result()
	if err != nil {
		return nil, false, err
	}
	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return nil, false, fmt.Errorf("unexpected Redis concurrency response: %T", result)
	}
	acquired, ok := redisInt64(values[0])
	if !ok {
		return nil, false, fmt.Errorf("invalid Redis concurrency acquired response: %v", values[0])
	}
	if acquired == 0 {
		return nil, false, nil
	}
	return newChannelConcurrencyLease(channelID, token, true, ctx), true, nil
}

func renewRedisChannelConcurrency(channelID int, token string) error {
	_, err := RDB.Eval(context.Background(), renewChannelConcurrencyScript, []string{channelConcurrencyKey(channelID)}, redisNowMilliseconds(), channelConcurrencyLeaseTTL.Milliseconds(), token).Result()
	return err
}

func releaseRedisChannelConcurrency(channelID int, token string) error {
	_, err := RDB.Eval(context.Background(), releaseChannelConcurrencyScript, []string{channelConcurrencyKey(channelID)}, token).Result()
	return err
}

func countRedisChannelConcurrency(ctx context.Context, channelID int) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := RDB.Eval(ctx, countChannelConcurrencyScript, []string{channelConcurrencyKey(channelID)}, redisNowMilliseconds()).Result()
	if err != nil {
		return 0, err
	}
	value, ok := redisInt64(result)
	if !ok {
		return 0, fmt.Errorf("invalid Redis concurrency count response: %v", result)
	}
	return int(value), nil
}

func redisInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case string:
		var parsed int64
		if _, err := fmt.Sscan(v, &parsed); err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}
