package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateChannelNormalizesZeroConcurrencyLimit(t *testing.T) {
	zero := 0
	channel := &model.Channel{ConcurrencyLimit: &zero}
	require.NoError(t, validateChannel(channel, false))
	assert.Nil(t, channel.ConcurrencyLimit)

	negative := -1
	channel.ConcurrencyLimit = &negative
	assert.ErrorContains(t, validateChannel(channel, false), "concurrency limit cannot be negative")
}

func TestGetChannelConcurrencyReturnsCurrentCountLimitAndKnownState(t *testing.T) {
	db := setupMultiKeyControllerTest(t)
	limit := 4
	channel := model.Channel{
		Type:             constant.ChannelTypeOpenAI,
		Key:              "key",
		Status:           common.ChannelStatusEnabled,
		Name:             "concurrency-api",
		Models:           "gpt-4",
		Group:            "default",
		ConcurrencyLimit: &limit,
	}
	require.NoError(t, db.Create(&channel).Error)

	lease, acquired, err := common.AcquireChannelConcurrency(context.Background(), channel.Id, &limit)
	require.NoError(t, err)
	require.True(t, acquired)
	defer lease.Release()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/channel/concurrency?ids="+strconv.Itoa(channel.Id), nil)
	GetChannelConcurrency(ctx)

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Items []struct {
				ChannelID          int  `json:"channel_id"`
				CurrentConcurrency int  `json:"current_concurrency"`
				ConcurrencyLimit   *int `json:"concurrency_limit"`
				Known              bool `json:"known"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Len(t, response.Data.Items, 1)
	item := response.Data.Items[0]
	assert.Equal(t, channel.Id, item.ChannelID)
	assert.Equal(t, 1, item.CurrentConcurrency)
	assert.Equal(t, limit, *item.ConcurrencyLimit)
	assert.True(t, item.Known)

}
