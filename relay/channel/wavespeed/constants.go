package wavespeed

const (
	// ChannelName identifies the WaveSpeed channel.
	ChannelName = "wavespeed"
	// ModelSeedreamV50Lite is the default image generation model supported by this channel.
	ModelSeedreamV50Lite = "bytedance/seedream-v5.0-lite"
)

var ModelList = []string{
	ModelSeedreamV50Lite,
}
