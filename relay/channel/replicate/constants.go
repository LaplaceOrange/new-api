package replicate

const (
	// ChannelName identifies the replicate channel.
	ChannelName = "replicate"
	// ModelFlux11Pro is the default image generation model supported by this channel.
	ModelFlux11Pro = "black-forest-labs/flux-1.1-pro"
	// ModelSeedreamV50Lite is a WaveSpeed-compatible image model.
	ModelSeedreamV50Lite = "bytedance/seedream-v5.0-lite"
)

var ModelList = []string{
	ModelFlux11Pro,
	ModelSeedreamV50Lite,
}
