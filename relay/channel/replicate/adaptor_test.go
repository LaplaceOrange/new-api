package replicate

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaveSpeedURL(t *testing.T) {
	assert.Equal(t, "https://api.wavespeed.ai/api/v3/bytedance/seedream-v5.0-lite", waveSpeedURL("https://api.wavespeed.ai", "/bytedance/seedream-v5.0-lite"))
	assert.Equal(t, "https://gateway.example/api/v3/bytedance/seedream-v5.0-lite", waveSpeedURL("https://gateway.example/api/v3", "/bytedance/seedream-v5.0-lite"))
}

func TestConvertImageRequestWaveSpeed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	a := &Adaptor{}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://api.wavespeed.ai"}}
	n := uint(1)
	request := dto.ImageRequest{
		Model:          "bytedance/seedream-v5.0-lite",
		Prompt:         "a cat",
		N:              &n,
		Size:           "2048x2048",
		ResponseFormat: "b64_json",
		OutputFormat:   []byte(`"jpeg"`),
	}

	converted, err := a.ConvertImageRequest(c, info, request)
	require.NoError(t, err)
	payload, ok := converted.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "a cat", payload["prompt"])
	assert.Equal(t, "2048*2048", payload["size"])
	assert.Equal(t, "jpeg", payload["output_format"])
	assert.Equal(t, true, payload["enable_base64_output"])
	assert.Equal(t, "/bytedance/seedream-v5.0-lite", info.RequestURLPath)
}

func TestConvertImageRequestWaveSpeedRejectsUnsupportedMultipleOutputs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	a := &Adaptor{}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://api.wavespeed.ai"}}
	n := uint(2)
	_, err := a.ConvertImageRequest(c, info, dto.ImageRequest{
		Model:  "bytedance/seedream-v5.0-lite",
		Prompt: "a cat",
		N:      &n,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not expose multiple image outputs")
}

func TestConvertImageRequestRejectsOversizedN(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	a := &Adaptor{}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://api.wavespeed.ai"}}
	n := uint(dto.MaxImageN + 1)
	_, err := a.ConvertImageRequest(c, info, dto.ImageRequest{Model: ModelSeedreamV50Lite, Prompt: "a cat", N: &n})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "between 1 and")
}

func TestDoResponseWaveSpeedCompletedBase64(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://api.wavespeed.ai"},
		Request:     &dto.ImageRequest{ResponseFormat: "b64_json"},
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"code":200,"data":{"id":"task-1","status":"completed","outputs":["YmFzZTY0"]}}`)),
	}

	usage, apiErr := (&Adaptor{}).DoResponse(c, resp, info)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	var imageResponse dto.ImageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &imageResponse))
	require.Len(t, imageResponse.Data, 1)
	assert.Equal(t, "YmFzZTY0", imageResponse.Data[0].B64Json)
	assert.Empty(t, imageResponse.Data[0].Url)
}
