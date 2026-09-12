package wavespeed

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type Adaptor struct{}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info == nil {
		return "", errors.New("wavespeed adaptor: relay info is nil")
	}
	if strings.TrimSpace(info.ChannelBaseUrl) == "" {
		info.ChannelBaseUrl = constant.ChannelBaseURLs[constant.ChannelTypeWaveSpeed]
	}
	if strings.TrimSpace(info.RequestURLPath) == "" {
		return info.ChannelBaseUrl, nil
	}
	return waveSpeedURL(info.ChannelBaseUrl, info.RequestURLPath), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	if info == nil {
		return errors.New("wavespeed adaptor: relay info is nil")
	}
	if strings.TrimSpace(info.ApiKey) == "" {
		return errors.New("wavespeed adaptor: api key is required")
	}
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("Authorization", "Bearer "+info.ApiKey)
	req.Set("Content-Type", "application/json")
	req.Set("Accept", "application/json")
	return nil
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	if info == nil {
		return nil, errors.New("wavespeed adaptor: relay info is nil")
	}
	if strings.TrimSpace(request.Prompt) == "" && c != nil {
		request.Prompt = c.PostForm("prompt")
	}
	if strings.TrimSpace(request.Prompt) == "" {
		return nil, errors.New("wavespeed adaptor: prompt is required")
	}
	modelName := strings.TrimSpace(info.UpstreamModelName)
	if modelName == "" {
		modelName = strings.TrimSpace(request.Model)
	}
	if modelName == "" {
		modelName = ModelSeedreamV50Lite
	}
	info.UpstreamModelName = modelName
	info.RequestURLPath = "/" + strings.TrimLeft(modelName, "/")

	payload := map[string]any{"prompt": request.Prompt}
	if size := strings.TrimSpace(request.Size); size != "" {
		payload["size"] = strings.NewReplacer("x", "*", "X", "*").Replace(size)
	}
	if len(request.OutputFormat) > 0 {
		var outputFormat string
		if err := common.Unmarshal(request.OutputFormat, &outputFormat); err == nil && strings.TrimSpace(outputFormat) != "" {
			payload["output_format"] = strings.TrimSpace(outputFormat)
		}
	}
	if imageN := lo.FromPtrOr(request.N, uint(0)); imageN > 0 {
		if imageN > dto.MaxImageN {
			return nil, fmt.Errorf("n must be an integer between 1 and %d", dto.MaxImageN)
		}
		if imageN > 1 {
			return nil, fmt.Errorf("wavespeed model %q does not expose multiple image outputs; set n=1", modelName)
		}
	}
	if info.RelayMode == relayconstant.RelayModeImagesEdits {
		return nil, errors.New("wavespeed adaptor: image edits are not supported by this channel")
	}
	if len(request.ExtraFields) > 0 {
		var extra map[string]any
		if err := common.Unmarshal(request.ExtraFields, &extra); err != nil {
			return nil, fmt.Errorf("wavespeed adaptor: failed to decode extra_fields: %w", err)
		}
		for key, value := range extra {
			payload[key] = value
		}
	}
	for key, raw := range request.Extra {
		if raw == nil {
			continue
		}
		if strings.EqualFold(key, "input") {
			var extraInput map[string]any
			if err := common.Unmarshal(raw, &extraInput); err != nil {
				return nil, fmt.Errorf("wavespeed adaptor: failed to decode extra input: %w", err)
			}
			for name, value := range extraInput {
				payload[name] = value
			}
			continue
		}
		var value any
		if err := common.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("wavespeed adaptor: failed to decode extra field %s: %w", key, err)
		}
		payload[key] = value
	}
	if strings.EqualFold(request.ResponseFormat, "b64_json") {
		payload["enable_base64_output"] = true
	}
	return payload, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (any, *types.NewAPIError) {
	if resp == nil {
		return nil, types.NewError(errors.New("wavespeed adaptor: empty response"), types.ErrorCodeBadResponse)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeReadResponseBodyFailed)
	}
	var envelope WaveSpeedResponse
	if err := common.Unmarshal(body, &envelope); err != nil {
		return nil, types.NewError(fmt.Errorf("wavespeed adaptor: decode response failed: %w", err), types.ErrorCodeBadResponseBody)
	}
	if envelope.Code >= 400 {
		message := strings.TrimSpace(envelope.Message)
		if message == "" {
			message = fmt.Sprintf("wavespeed request failed with code %d", envelope.Code)
		}
		return nil, types.NewError(errors.New(message), types.ErrorCodeBadResponse)
	}
	prediction := envelope.Data
	if prediction.ID == "" {
		if err := common.Unmarshal(body, &prediction); err != nil {
			return nil, types.NewError(fmt.Errorf("wavespeed adaptor: missing prediction id: %w", err), types.ErrorCodeBadResponseBody)
		}
	}
	if prediction.ID == "" {
		return nil, types.NewError(errors.New("wavespeed adaptor: submission response missing prediction id"), types.ErrorCodeBadResponseBody)
	}
	if !isTerminal(prediction.Status) {
		resultURL := strings.TrimSpace(prediction.URLs.Get)
		prediction, err = poll(c, info, prediction.ID, resultURL)
		if err != nil {
			return nil, types.NewError(err, types.ErrorCodeBadResponse)
		}
	}
	if !strings.EqualFold(prediction.Status, "completed") {
		status := strings.TrimSpace(prediction.Status)
		if status == "" {
			status = "unknown"
		}
		return nil, types.NewError(fmt.Errorf("wavespeed prediction status %q: %v", status, prediction.Error), types.ErrorCodeBadResponse)
	}
	if len(prediction.Outputs) == 0 {
		return nil, types.NewError(errors.New("wavespeed adaptor: empty prediction outputs"), types.ErrorCodeBadResponseBody)
	}
	wantsBase64 := false
	if imageReq, ok := info.Request.(*dto.ImageRequest); ok && imageReq != nil {
		wantsBase64 = strings.EqualFold(imageReq.ResponseFormat, "b64_json")
	}
	imageResponse := dto.ImageResponse{Created: common.GetTimestamp(), Data: make([]dto.ImageData, 0, len(prediction.Outputs))}
	for _, output := range prediction.Outputs {
		value, kind := outputValue(output)
		if value == "" {
			continue
		}
		if wantsBase64 && kind == "url" {
			_, value, err = service.GetImageFromUrl(value)
			if err != nil {
				return nil, types.NewError(fmt.Errorf("wavespeed adaptor: download image failed: %w", err), types.ErrorCodeBadResponse)
			}
			kind = "base64"
		}
		if kind == "base64" {
			imageResponse.Data = append(imageResponse.Data, dto.ImageData{B64Json: value})
		} else {
			imageResponse.Data = append(imageResponse.Data, dto.ImageData{Url: value})
		}
	}
	if len(imageResponse.Data) == 0 {
		return nil, types.NewError(errors.New("wavespeed adaptor: no usable image outputs"), types.ErrorCodeBadResponseBody)
	}
	responseBytes, err := common.Marshal(imageResponse)
	if err != nil {
		return nil, types.NewError(fmt.Errorf("wavespeed adaptor: encode response failed: %w", err), types.ErrorCodeBadResponseBody)
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.Write(responseBytes)
	return &dto.Usage{}, nil
}

func waveSpeedURL(baseURL, path string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	path = "/" + strings.TrimLeft(path, "/")
	if strings.HasSuffix(strings.ToLower(base), "/api/v3") {
		return base + path
	}
	return base + "/api/v3" + path
}

func isTerminal(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "failed", "cancelled", "canceled", "timeout", "deleted":
		return true
	default:
		return false
	}
}

func poll(c *gin.Context, info *relaycommon.RelayInfo, predictionID, resultURL string) (WaveSpeedPrediction, error) {
	if info == nil {
		return WaveSpeedPrediction{}, errors.New("wavespeed adaptor: relay info is nil")
	}
	if strings.TrimSpace(resultURL) == "" {
		baseURL := info.ChannelBaseUrl
		if strings.TrimSpace(baseURL) == "" {
			baseURL = constant.ChannelBaseURLs[constant.ChannelTypeWaveSpeed]
		}
		resultURL = waveSpeedURL(baseURL, "/predictions/"+predictionID+"/result")
	}
	client, err := service.GetHttpClientWithProxySettings(info.ChannelSetting.Proxy, info.ChannelSetting)
	if err != nil {
		return WaveSpeedPrediction{}, fmt.Errorf("wavespeed adaptor: create http client failed: %w", err)
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	deadline := time.NewTimer(10 * time.Minute)
	defer deadline.Stop()
	for {
		select {
		case <-ticker.C:
			prediction, err := fetch(c, client, resultURL, info)
			if err != nil {
				return WaveSpeedPrediction{}, err
			}
			if isTerminal(prediction.Status) {
				return prediction, nil
			}
		case <-deadline.C:
			return WaveSpeedPrediction{}, errors.New("wavespeed adaptor: prediction polling timed out")
		case <-c.Request.Context().Done():
			return WaveSpeedPrediction{}, c.Request.Context().Err()
		}
	}
}

func fetch(c *gin.Context, client *http.Client, resultURL string, info *relaycommon.RelayInfo) (WaveSpeedPrediction, error) {
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, resultURL, nil)
	if err != nil {
		return WaveSpeedPrediction{}, fmt.Errorf("wavespeed adaptor: create result request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)
	req.Header.Set("Accept", "application/json")
	if overrides, err := channel.ResolveHeaderOverride(info, c); err != nil {
		return WaveSpeedPrediction{}, fmt.Errorf("wavespeed adaptor: resolve header override failed: %w", err)
	} else {
		for key, value := range overrides {
			req.Header.Set(key, value)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return WaveSpeedPrediction{}, fmt.Errorf("wavespeed adaptor: fetch result failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WaveSpeedPrediction{}, fmt.Errorf("wavespeed adaptor: read result failed: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return WaveSpeedPrediction{}, fmt.Errorf("wavespeed adaptor: result request returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var envelope WaveSpeedResponse
	if err := common.Unmarshal(body, &envelope); err != nil {
		return WaveSpeedPrediction{}, fmt.Errorf("wavespeed adaptor: decode result failed: %w", err)
	}
	if envelope.Code >= 400 {
		return WaveSpeedPrediction{}, fmt.Errorf("wavespeed adaptor: result failed with code %d: %s", envelope.Code, envelope.Message)
	}
	if envelope.Data.ID != "" || envelope.Data.Status != "" {
		return envelope.Data, nil
	}
	var prediction WaveSpeedPrediction
	if err := common.Unmarshal(body, &prediction); err != nil {
		return WaveSpeedPrediction{}, fmt.Errorf("wavespeed adaptor: decode prediction failed: %w", err)
	}
	return prediction, nil
}

func outputValue(output any) (string, string) {
	switch value := output.(type) {
	case string:
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
			return value, "url"
		}
		return value, "base64"
	case map[string]any:
		for _, key := range []string{"url", "image_url", "b64_json", "base64"} {
			if raw, ok := value[key].(string); ok && strings.TrimSpace(raw) != "" {
				kind := "url"
				if key == "b64_json" || key == "base64" {
					kind = "base64"
				}
				return strings.TrimSpace(raw), kind
			}
		}
	}
	return "", ""
}

func (a *Adaptor) GetModelList() []string { return ModelList }
func (a *Adaptor) GetChannelName() string { return ChannelName }

func (a *Adaptor) ConvertOpenAIRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeneralOpenAIRequest) (any, error) {
	return nil, errors.New("wavespeed adaptor: ConvertOpenAIRequest is not implemented")
}
func (a *Adaptor) ConvertRerankRequest(*gin.Context, int, dto.RerankRequest) (any, error) {
	return nil, errors.New("wavespeed adaptor: ConvertRerankRequest is not implemented")
}
func (a *Adaptor) ConvertEmbeddingRequest(*gin.Context, *relaycommon.RelayInfo, dto.EmbeddingRequest) (any, error) {
	return nil, errors.New("wavespeed adaptor: ConvertEmbeddingRequest is not implemented")
}
func (a *Adaptor) ConvertAudioRequest(*gin.Context, *relaycommon.RelayInfo, dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("wavespeed adaptor: ConvertAudioRequest is not implemented")
}
func (a *Adaptor) ConvertOpenAIResponsesRequest(*gin.Context, *relaycommon.RelayInfo, dto.OpenAIResponsesRequest) (any, error) {
	return nil, errors.New("wavespeed adaptor: ConvertOpenAIResponsesRequest is not implemented")
}
func (a *Adaptor) ConvertClaudeRequest(*gin.Context, *relaycommon.RelayInfo, *dto.ClaudeRequest) (any, error) {
	return nil, errors.New("wavespeed adaptor: ConvertClaudeRequest is not implemented")
}
func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("wavespeed adaptor: ConvertGeminiRequest is not implemented")
}
