package controller

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelModelStatusDatabaseMatrix(t *testing.T) {
	for _, dialect := range []struct{ kind, env string }{
		{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"},
	} {
		t.Run(dialect.kind, func(t *testing.T) {
			if dialect.env != "" && os.Getenv(dialect.env) == "" {
				t.Skip("set " + dialect.env + " to run this database")
			}
			db := modelManagementDB(t, dialect.kind, os.Getenv(dialect.env))
			for range 2 {
				require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}))
			}
			tag := "model-status"
			first := model.Channel{
				Type: 1, Key: "test-key", Name: "first", Models: "a,b", Group: "default,other",
				Status: common.ChannelStatusEnabled, Tag: &tag,
			}
			second := first
			second.Name = "second"
			require.NoError(t, first.Insert())
			require.NoError(t, second.Insert())

			request := func(body string) bool {
				recorder := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(recorder)
				ctx.Set("role", common.RoleRootUser)
				ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(first.Id)}}
				ctx.Request = httptest.NewRequest(http.MethodPost, "/api/channel/"+strconv.Itoa(first.Id)+"/models/status", bytes.NewBufferString(body))
				UpdateChannelModelsStatus(ctx)
				var response struct {
					Success bool `json:"success"`
				}
				require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
				assert.NotContains(t, recorder.Body.String(), "test-key")
				return response.Success
			}

			for _, body := range []string{
				`{"models":["a"]}`, `{"models":[],"enabled":false}`,
				`{"models":["a","absent"],"enabled":false}`,
				`{"models":[""],"enabled":false}`, `{"models":["a"],"enabled":"false"}`,
			} {
				assert.False(t, request(body), body)
			}
			assert.True(t, model.IsChannelEnabledForGroupModel("default", "a", first.Id))
			require.True(t, request(`{"models":["a"],"enabled":false}`))
			var stored model.Channel
			require.NoError(t, db.First(&stored, first.Id).Error)
			assert.Equal(t, []string{"a", "b"}, stored.GetModels(), "disabled models remain testable")
			assert.True(t, stored.ChannelInfo.DisabledModels["a"])
			assert.False(t, model.IsChannelEnabledForGroupModel("default", "a", first.Id))
			assert.False(t, model.IsChannelEnabledForGroupModel("other", "a", first.Id))
			assert.True(t, model.IsChannelEnabledForGroupModel("default", "a", second.Id))
			assert.True(t, model.IsChannelEnabledForGroupModel("default", "b", first.Id))

			for _, memoryCache := range []bool{false, true} {
				common.MemoryCacheEnabled = memoryCache
				model.InitChannelCache()
				selected, lease, err := model.GetRandomSatisfiedChannelWithConcurrency(context.Background(), "default", "a", "")
				require.NoError(t, err)
				require.NotNil(t, selected)
				require.NotNil(t, lease)
				lease.Release()
				assert.Equal(t, second.Id, selected.Id)
				allowed, _ := model.ChannelSatisfiesFilters(&stored, "a", nil)
				assert.False(t, allowed, "pinned and retry paths reject disabled models")
			}
			common.MemoryCacheEnabled = false
			require.True(t, model.UpdateChannelStatus(first.Id, "", common.ChannelStatusManuallyDisabled, "test"))
			require.True(t, model.UpdateChannelStatus(first.Id, "", common.ChannelStatusEnabled, "test"))
			require.NoError(t, model.EnableChannelByTag(tag))
			require.NoError(t, model.UpdateAbilityStatus(first.Id, true))
			require.NoError(t, model.UpdateAbilityStatusByTag(tag, true))
			_, _, err := model.FixAbility()
			require.NoError(t, err)
			assert.False(t, model.IsChannelEnabledForGroupModel("default", "a", first.Id))

			require.True(t, request(`{"models":["a"],"enabled":true}`))
			assert.True(t, model.IsChannelEnabledForGroupModel("default", "a", first.Id))
			// A model toggle must never implicitly enable an entire disabled channel.
			require.True(t, model.UpdateChannelStatus(first.Id, "", common.ChannelStatusManuallyDisabled, "test"))
			require.True(t, request(`{"models":["a","b"],"enabled":false}`))
			require.True(t, request(`{"models":["a","b"],"enabled":true}`))
			assert.False(t, model.IsChannelEnabledForGroupModel("default", "a", first.Id))
		})
	}
}
