package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkbuddyCapacityAdmissionAndFallback(t *testing.T) {
	for index, scenario := range []struct {
		name     string
		status   int
		capacity string
		occupy   bool
	}{
		{"available", 200, `{"model_available_accounts":1,"available_accounts":2}`, false},
		{"model full", 200, `{"model_available_accounts":1,"available_accounts":2}`, true},
		{"zero capacity", 200, `{"model_available_accounts":0,"available_accounts":2}`, false},
		{"upstream failure", 503, `{}`, false},
		{"redirect rejected", 302, `{}`, false},
		{"invalid counts", 200, `{"model_available_accounts":3,"available_accounts":2}`, false},
		{"malformed response", 200, `invalid`, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db := setupChannelSelectAutoGroupsTest(t)
			oldRedis := common.RedisEnabled
			common.RedisEnabled = false
			t.Cleanup(func() { common.RedisEnabled = oldRedis })
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				assert.Equal(t, "/v1/capacity", r.URL.Path)
				assert.Equal(t, "upstream-model", r.URL.Query().Get("model"))
				assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
				if scenario.status == http.StatusFound {
					w.Header().Set("Location", "/redirected")
				}
				w.WriteHeader(scenario.status)
				_, _ = w.Write([]byte(scenario.capacity))
			}))
			t.Cleanup(server.Close)
			id := 7300 + index*10
			createChannelSelectAutoGroupsChannel(t, db, id, "default", "alias")
			createChannelSelectAutoGroupsChannel(t, db, id+1, "default", "alias")
			perAccount := 1
			mapping := `{"alias":"upstream-model","another-alias":"upstream-model"}`
			require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", id).Updates(map[string]any{
				"type": constant.ChannelTypeWorkbuddy2API, "base_url": server.URL,
				"key": "test-key", "model_mapping": mapping, "concurrency_limit": perAccount, "priority": 10,
			}).Error)
			model.InitChannelCache()
			candidate := &model.Channel{Id: id, Type: constant.ChannelTypeWorkbuddy2API,
				BaseURL: &server.URL, Key: "test-key", ModelMapping: &mapping, ConcurrencyLimit: &perAccount}
			if scenario.occupy {
				lease, acquired, err := AcquireWorkbuddy2APIConcurrency(context.Background(), candidate, "another-alias", "test-key")
				require.NoError(t, err)
				require.True(t, acquired)
				t.Cleanup(lease.Release)
			}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			selected, _, err := CacheGetRandomSatisfiedChannel(&RetryParam{Ctx: c, TokenGroup: "default", ModelName: "alias", RequestPath: "/v1/chat/completions"})
			require.NoError(t, err)
			require.NotNil(t, selected)
			if scenario.name == "available" {
				assert.Equal(t, id, selected.Id)
				assert.Equal(t, "test-key", common.GetContextKeyString(c, constant.ContextKeyChannelPreselectedKey))
			} else {
				assert.Equal(t, id+1, selected.Id)
			}
			ReleaseChannelConcurrency(c)
			count, known, err := common.GetChannelConcurrency(context.Background(), selected.Id)
			require.NoError(t, err)
			assert.True(t, known)
			assert.Zero(t, count)
			assert.EqualValues(t, 1, calls.Load(), "aliases share the one-second upstream-model capacity cache")
		})
	}
}

func TestWorkbuddyChannelLimitAppliesAcrossModels(t *testing.T) {
	oldRedis := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = oldRedis })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"model_available_accounts":2,"available_accounts":2}`))
	}))
	t.Cleanup(server.Close)
	perAccount := 1
	candidate := &model.Channel{Id: 7401, Type: constant.ChannelTypeWorkbuddy2API, BaseURL: &server.URL, ConcurrencyLimit: &perAccount}
	first, ok, err := AcquireWorkbuddy2APIConcurrency(context.Background(), candidate, "model-a", "key")
	require.NoError(t, err)
	require.True(t, ok)
	defer first.Release()
	second, ok, err := AcquireWorkbuddy2APIConcurrency(context.Background(), candidate, "model-b", "key")
	require.NoError(t, err)
	require.True(t, ok)
	defer second.Release()
	_, ok, err = AcquireWorkbuddy2APIConcurrency(context.Background(), candidate, "model-c", "key")
	require.NoError(t, err)
	assert.False(t, ok)
	first.Release()
	replacement, ok, err := AcquireWorkbuddy2APIConcurrency(context.Background(), candidate, "model-c", "key")
	require.NoError(t, err)
	require.True(t, ok)
	replacement.Release()
	second.Release()
	count, _, err := common.GetChannelConcurrency(context.Background(), candidate.Id)
	require.NoError(t, err)
	assert.Zero(t, count)
}

func TestPinnedTaskPluginChannelTypesUsesPinnedGenerationIndex(t *testing.T) {
	registry := jsplugin.NewRegistry()
	plugin, err := registry.Register(channelSelectTaskPluginSource("legacy-select", constant.ChannelTypeKling), jsplugin.Options{})
	require.NoError(t, err)

	c, _ := gin.CreateTestContext(nil)
	c.Set(jsplugin.ContextKeyPinnedPlugin, jsplugin.PinnedPlugin{
		Generation: registry.Generation(),
		Plugin:     plugin,
	})

	types, keys := pinnedTaskPluginIdentities(c, "legacy-select")
	assert.Equal(t, []int{constant.ChannelTypeKling}, types)
	assert.Equal(t, []string{"legacy-select"}, keys)
	types, keys = pinnedTaskPluginIdentities(c, "another-plugin")
	assert.Empty(t, types)
	assert.Empty(t, keys)
	types, keys = pinnedTaskPluginIdentities(nil, "legacy-select")
	assert.Empty(t, types)
	assert.Empty(t, keys)
}

func TestPinnedTaskPluginChannelTypesLeavesGenericChannelsKeyed(t *testing.T) {
	registry := jsplugin.NewRegistry()
	plugin, err := registry.Register(channelSelectTaskPluginSource("generic-select", constant.ChannelTypeTaskPlugin), jsplugin.Options{})
	require.NoError(t, err)

	c, _ := gin.CreateTestContext(nil)
	c.Set(jsplugin.ContextKeyPinnedPlugin, jsplugin.PinnedPlugin{
		Generation: registry.Generation(),
		Plugin:     plugin,
	})

	types, keys := pinnedTaskPluginIdentities(c, "generic-select")
	assert.Empty(t, types)
	assert.Equal(t, []string{"generic-select"}, keys)
}

func TestPinnedTaskPluginChannelTypesIncludesSharedEndpointProviders(t *testing.T) {
	registry := jsplugin.NewRegistry()
	_, err := registry.Register(channelSelectEndpointPluginSource("gemini-select", constant.ChannelTypeGemini), jsplugin.Options{})
	require.NoError(t, err)
	_, err = registry.Register(channelSelectEndpointPluginSource("vertex-select", constant.ChannelTypeVertexAi), jsplugin.Options{})
	require.NoError(t, err)
	candidates := registry.Generation().LookupEndpointCandidates("POST", "/v1/responses", "task-model")
	require.Len(t, candidates, 2)

	c, _ := gin.CreateTestContext(nil)
	c.Set(jsplugin.ContextKeyPinnedPlugin, jsplugin.PinnedPlugin{
		Generation: registry.Generation(),
		Plugin:     candidates[0].Plugin,
	})
	c.Set(jsplugin.ContextKeyPinnedEndpoint, jsplugin.PinnedEndpoint{
		Generation: registry.Generation(),
		Plugin:     candidates[0].Plugin,
		Protocol:   candidates[0].Protocol,
		Operation:  candidates[0].Operation,
		Model:      "task-model",
		Candidates: candidates,
	})

	AppendTaskPluginIdentityFilter(c, candidates[0].Plugin.Meta.Key)
	filters := GetChannelConstraints(c).Filters
	require.Len(t, filters, 1)
	assert.Equal(t, []int{constant.ChannelTypeGemini, constant.ChannelTypeVertexAi}, filters[0].TaskPluginChannelTypes)
	assert.Equal(t, []string{"gemini-select", "vertex-select"}, filters[0].TaskPluginKeys)
}

func channelSelectTaskPluginSource(key string, channelType int) string {
	return fmt.Sprintf(`
export const meta = {
  apiVersion: 1,
  key: %q,
  name: %q,
  version: "1.0.0",
  author: {name: "Test"},
  %s
  models: ["task-model"],
  fetchMode: "per_task",
};
export function buildSubmitRequest() { return {}; }
export function parseSubmitResponse() { return {taskId: "task"}; }
export function buildQueryRequest() { return {}; }
export function parseTaskResult() { return {status: "SUCCESS"}; }
`, key, key, channelSelectChannelTypesField(channelType))
}

func channelSelectEndpointPluginSource(key string, channelType int) string {
	return fmt.Sprintf(`
export const meta = {
  apiVersion: 1,
  key: %q,
  name: %q,
  version: "1.0.0",
  author: {name: "Test"},
  %s
  models: ["task-model"],
  fetchMode: "per_task",
  protocols: [{name: "openai_responses", supports: ["stream", "sync", "background"]}],
};
export function buildSubmitRequest() { return {}; }
export function parseSubmitResponse() { return {taskId: "task"}; }
export function buildQueryRequest() { return {}; }
export function parseTaskResult() { return {status: "SUCCESS"}; }
export const protocols = {openai_responses: {
  decodeRequest: function(ctx) { return {kind: "submit", model: "task-model", requestBody: ctx.body.value}; },
  renderEvents: function() { return {events: [], state: null, done: false}; },
  renderFinal: function() { return {output: []}; },
}};
`, key, key, channelSelectChannelTypesField(channelType))
}

func channelSelectChannelTypesField(channelType int) string {
	if channelType <= 0 || channelType == constant.ChannelTypeTaskPlugin {
		return ""
	}
	return fmt.Sprintf("channelTypes: [%d],", channelType)
}

func TestPinnedTaskPluginChannelTypesIncludesCompatibleTypes(t *testing.T) {
	registry := jsplugin.NewRegistry()
	plugin, err := registry.Register(channelSelectCompatiblePluginSource("sora-select", constant.ChannelTypeSora, constant.ChannelTypeOpenAI), jsplugin.Options{})
	require.NoError(t, err)

	c, _ := gin.CreateTestContext(nil)
	c.Set(jsplugin.ContextKeyPinnedPlugin, jsplugin.PinnedPlugin{
		Generation: registry.Generation(),
		Plugin:     plugin,
	})

	types, keys := pinnedTaskPluginIdentities(c, "sora-select")
	assert.Equal(t, []int{constant.ChannelTypeSora, constant.ChannelTypeOpenAI}, types)
	assert.Equal(t, []string{"sora-select"}, keys)
}

func channelSelectCompatiblePluginSource(key string, channelType, compatibleType int) string {
	return fmt.Sprintf(`
export const meta = {
  apiVersion: 1,
  key: %q,
  name: %q,
  version: "1.0.0",
  author: {name: "Test"},
  channelTypes: [%d, %d],
  models: ["task-model"],
  fetchMode: "per_task",
};
export function buildSubmitRequest() { return {}; }
export function parseSubmitResponse() { return {taskId: "task"}; }
export function buildQueryRequest() { return {}; }
export function parseTaskResult() { return {status: "SUCCESS"}; }
`, key, key, channelType, compatibleType)
}

func TestSharedType61IdentityFilterContainsAllCandidateKeys(t *testing.T) {
	registry := jsplugin.NewRegistry()
	for _, key := range []string{"alpha", "beta"} {
		_, err := registry.Register(channelSelectEndpointPluginSource(key, 0), jsplugin.Options{})
		require.NoError(t, err)
	}
	generation := registry.Generation()
	candidates := generation.LookupEndpointCandidates("POST", "/v1/responses", "task-model")
	require.Len(t, candidates, 2)
	c, _ := gin.CreateTestContext(nil)
	c.Set(jsplugin.ContextKeyPinnedEndpoint, jsplugin.PinnedEndpoint{Generation: generation, Plugin: candidates[0].Plugin, Candidates: candidates})
	AppendTaskPluginIdentityFilter(c, "alpha")
	filters := GetChannelConstraints(c).Filters
	require.Len(t, filters, 1)
	assert.Equal(t, "alpha", filters[0].TaskPluginKey)
	assert.Equal(t, []string{"alpha", "beta"}, filters[0].TaskPluginKeys)
	assert.Empty(t, filters[0].TaskPluginChannelTypes)
}
