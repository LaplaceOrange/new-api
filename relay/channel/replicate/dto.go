package replicate

type PredictionResponse struct {
	Status string           `json:"status"`
	Output any              `json:"output"`
	Error  *PredictionError `json:"error"`
}

// WaveSpeedResponse is the envelope returned by WaveSpeed's v3 API.
// The provider may omit the envelope for some gateway deployments, so the
// adaptor also accepts a prediction object directly when decoding responses.
type WaveSpeedResponse struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Data    WaveSpeedPrediction `json:"data"`
}

type WaveSpeedPrediction struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Outputs []any  `json:"outputs"`
	Error   any    `json:"error"`
	URLs    struct {
		Get string `json:"get"`
	} `json:"urls"`
}

type PredictionError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

type FileUploadResponse struct {
	Urls struct {
		Get string `json:"get"`
	} `json:"urls"`
}
