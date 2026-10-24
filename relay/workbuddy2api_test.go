package relay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkbuddyConversationEndpointsAlwaysSendChatCompletions(t *testing.T) {
	original := model_setting.GetGlobalSettings().PassThroughRequestEnabled
	model_setting.GetGlobalSettings().PassThroughRequestEnabled = true
	t.Cleanup(func() { model_setting.GetGlobalSettings().PassThroughRequestEnabled = original })
	for _, endpoint := range []struct {
		path    string
		format  types.RelayFormat
		body    string
		request dto.Request
	}{
		{"/v1/chat/completions", types.RelayFormatOpenAI, `{"model":"alias","messages":[{"role":"user","content":"hello"}]}`, &dto.GeneralOpenAIRequest{}},
		{"/v1/messages", types.RelayFormatClaude, `{"model":"alias","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`, &dto.ClaudeRequest{}},
		{"/v1/responses", types.RelayFormatOpenAIResponses, `{"model":"alias","input":"hello"}`, &dto.OpenAIResponsesRequest{}},
	} {
		t.Run(endpoint.path, func(t *testing.T) {
			var captured map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/chat/completions", r.URL.Path)
				assert.Equal(t, "Bearer key", r.Header.Get("Authorization"))
				require.NoError(t, common.DecodeJson(r.Body, &captured))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":{"message":"test upstream error","type":"invalid_request_error"}}`)
			}))
			t.Cleanup(server.Close)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, endpoint.path, strings.NewReader(endpoint.body))
			c.Request.Header.Set("Content-Type", "application/json")
			require.NoError(t, common.UnmarshalJsonStr(endpoint.body, endpoint.request))
			common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeWorkbuddy2API)
			common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, server.URL)
			common.SetContextKey(c, constant.ContextKeyChannelKey, "key")
			common.SetContextKey(c, constant.ContextKeyOriginalModel, "alias")
			common.SetContextKey(c, constant.ContextKeyChannelSetting, dto.ChannelSettings{PassThroughBodyEnabled: true})
			c.Set("model_mapping", `{"alias":"upstream-model"}`)
			info := &relaycommon.RelayInfo{
				RelayFormat: endpoint.format, RelayMode: relayconstant.Path2RelayMode(endpoint.path),
				OriginModelName: "alias", Request: endpoint.request,
			}
			var apiErr *types.NewAPIError
			switch endpoint.format {
			case types.RelayFormatClaude:
				apiErr = ClaudeHelper(c, info)
			case types.RelayFormatOpenAIResponses:
				apiErr = ResponsesHelper(c, info)
			default:
				apiErr = TextHelper(c, info)
			}
			require.NotNil(t, apiErr)
			assert.Equal(t, 400, apiErr.StatusCode)
			require.NotNil(t, captured)
			assert.Equal(t, "upstream-model", captured["model"])
			assert.Contains(t, captured, "messages")
			assert.NotContains(t, captured, "input")
			assert.False(t, info.IsRequestBodyPassthroughEnabled())
		})
	}
}

func TestWorkbuddyConversationResponseFormats(t *testing.T) {
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })
	for _, format := range []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude, types.RelayFormatOpenAIResponses} {
		for _, stream := range []bool{false, true} {
			t.Run(string(format)+map[bool]string{false: "/json", true: "/stream"}[stream], func(t *testing.T) {
				rec := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(rec)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
				mode := relayconstant.RelayModeChatCompletions
				if format == types.RelayFormatOpenAIResponses {
					mode = relayconstant.RelayModeResponses
				}
				info := &relaycommon.RelayInfo{RelayMode: mode, RelayFormat: format, IsStream: stream, DisablePing: true,
					OriginModelName: "model", Request: &dto.GeneralOpenAIRequest{Model: "model"},
					ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeWorkbuddy2API, UpstreamModelName: "model"}}
				adaptor := GetAdaptor(constant.APITypeWorkbuddy2API)
				require.NotNil(t, adaptor)
				adaptor.Init(info)
				body := `{"id":"chatcmpl-1","object":"chat.completion","model":"model","choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`
				contentType := "application/json"
				if stream {
					contentType = "text/event-stream"
					body = "data: " + `{"id":"chatcmpl-1","object":"chat.completion.chunk","model":"model","choices":[{"index":0,"delta":{"role":"assistant","content":"hello"},"finish_reason":null}]}` + "\n\ndata: " + `{"id":"chatcmpl-1","object":"chat.completion.chunk","model":"model","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}` + "\n\ndata: [DONE]\n\n"
				}
				response := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(body))}
				usage, apiErr := adaptor.DoResponse(c, response, info)
				require.Nil(t, apiErr)
				require.NotNil(t, usage)
				tokens, ok := usage.(*dto.Usage)
				require.True(t, ok)
				assert.Equal(t, 3, tokens.PromptTokens)
				assert.Equal(t, 1, tokens.CompletionTokens)
				assert.Equal(t, 4, tokens.TotalTokens)
				assert.Contains(t, rec.Body.String(), "hello")
				switch format {
				case types.RelayFormatClaude:
					if stream {
						assert.Contains(t, rec.Body.String(), "message_stop")
					} else {
						assert.Contains(t, rec.Body.String(), `"type":"message"`)
					}
				case types.RelayFormatOpenAIResponses:
					if stream {
						assert.Contains(t, rec.Body.String(), "response.completed")
					} else {
						assert.Contains(t, rec.Body.String(), `"object":"response"`)
					}
				default:
					if stream {
						assert.Contains(t, rec.Body.String(), "[DONE]")
					} else {
						assert.Contains(t, rec.Body.String(), `"object":"chat.completion"`)
					}
				}
			})
		}
	}
}
