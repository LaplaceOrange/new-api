package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	hostreasoning "github.com/QuantumNous/new-api/setting/reasoning"
	"golang.org/x/sync/singleflight"
)

type workbuddyCapacity struct {
	ModelAvailableAccounts int `json:"model_available_accounts"`
	AvailableAccounts      int `json:"available_accounts"`
}

type workbuddyCapacityCacheEntry struct {
	expiresAt time.Time
	value     workbuddyCapacity
	err       error
}

var workbuddyCapacityCache = struct {
	sync.Mutex
	entries map[string]workbuddyCapacityCacheEntry
}{entries: make(map[string]workbuddyCapacityCacheEntry)}
var workbuddyCapacityFlights singleflight.Group

func AcquireWorkbuddy2APIConcurrency(ctx context.Context, channel *model.Channel, requestModel, apiKey string) (*common.ChannelConcurrencyLease, bool, error) {
	if channel == nil {
		return nil, false, nil
	}
	if channel.Type != constant.ChannelTypeWorkbuddy2API {
		return common.AcquireChannelConcurrency(ctx, channel.Id, channel.ConcurrencyLimit)
	}
	limit := channel.ConcurrencyLimit
	if limit == nil || *limit <= 0 {
		return nil, false, fmt.Errorf("Workbuddy2API per-account concurrency limit must be a positive integer")
	}
	mappedModel, err := relaycommon.ResolveChannelModel(requestModel, channel.GetModelMapping())
	if err != nil {
		return nil, false, err
	}
	requestModel = hostreasoning.BaseModelName(mappedModel)
	capacity, err := getWorkbuddyCapacity(ctx, channel, requestModel, apiKey)
	if err != nil || capacity.ModelAvailableAccounts <= 0 || capacity.AvailableAccounts <= 0 {
		return nil, false, err
	}
	perAccountLimit := *limit
	maxInt := int(^uint(0) >> 1)
	if capacity.ModelAvailableAccounts > maxInt/perAccountLimit || capacity.AvailableAccounts > maxInt/perAccountLimit {
		return nil, false, fmt.Errorf("workbuddy capacity exceeds supported concurrency range")
	}
	modelLimit := capacity.ModelAvailableAccounts * perAccountLimit
	channelLimit := capacity.AvailableAccounts * perAccountLimit
	return common.AcquireChannelAndModelConcurrency(ctx, channel.Id, requestModel, &channelLimit, &modelLimit)
}

func getWorkbuddyCapacity(ctx context.Context, channel *model.Channel, requestModel, apiKey string) (workbuddyCapacity, error) {
	keyHash := sha256.Sum256([]byte(apiKey))
	key := fmt.Sprintf("%d\x00%s\x00%x", channel.Id, requestModel, keyHash)
	now := time.Now()
	workbuddyCapacityCache.Lock()
	if entry, ok := workbuddyCapacityCache.entries[key]; ok && now.Before(entry.expiresAt) {
		workbuddyCapacityCache.Unlock()
		return entry.value, entry.err
	}
	workbuddyCapacityCache.Unlock()

	result, err, _ := workbuddyCapacityFlights.Do(key, func() (any, error) {
		now := time.Now()
		workbuddyCapacityCache.Lock()
		if entry, ok := workbuddyCapacityCache.entries[key]; ok && now.Before(entry.expiresAt) {
			workbuddyCapacityCache.Unlock()
			return entry.value, entry.err
		}
		workbuddyCapacityCache.Unlock()

		capacity, err := fetchWorkbuddyCapacity(ctx, channel, requestModel, apiKey)
		now = time.Now()
		workbuddyCapacityCache.Lock()
		for cacheKey, entry := range workbuddyCapacityCache.entries {
			if !now.Before(entry.expiresAt) {
				delete(workbuddyCapacityCache.entries, cacheKey)
			}
		}
		workbuddyCapacityCache.entries[key] = workbuddyCapacityCacheEntry{
			expiresAt: now.Add(time.Second),
			value:     capacity,
			err:       err,
		}
		workbuddyCapacityCache.Unlock()
		return capacity, err
	})
	if err != nil {
		return workbuddyCapacity{}, err
	}
	capacity, ok := result.(workbuddyCapacity)
	if !ok {
		return workbuddyCapacity{}, fmt.Errorf("unexpected workbuddy capacity response %T", result)
	}
	return capacity, nil
}

func fetchWorkbuddyCapacity(ctx context.Context, channel *model.Channel, requestModel, apiKey string) (workbuddyCapacity, error) {
	baseURL := strings.TrimRight(channel.GetBaseURL(), "/")
	endpoint := baseURL + "/v1/capacity?model=" + url.QueryEscape(requestModel)
	requestContext, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestContext, http.MethodGet, endpoint, nil)
	if err != nil {
		return workbuddyCapacity{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	settings := channel.GetSetting()
	client, err := GetHttpClientWithProxySettings(settings.Proxy, settings)
	if err != nil {
		return workbuddyCapacity{}, err
	}
	// Never forward the channel credential to a redirected endpoint.
	capacityClient := *client
	capacityClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := capacityClient.Do(req)
	if err != nil {
		return workbuddyCapacity{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return workbuddyCapacity{}, fmt.Errorf("workbuddy capacity returned HTTP %d", resp.StatusCode)
	}
	var capacity workbuddyCapacity
	if err := common.DecodeJson(io.LimitReader(resp.Body, 1<<20), &capacity); err != nil {
		return workbuddyCapacity{}, err
	}
	if capacity.ModelAvailableAccounts < 0 || capacity.AvailableAccounts < capacity.ModelAvailableAccounts {
		return workbuddyCapacity{}, fmt.Errorf("workbuddy capacity returned inconsistent account counts")
	}
	return capacity, nil
}
