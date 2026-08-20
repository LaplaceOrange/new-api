package dto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneralOpenAIRequestSensitiveTextExcludesProtocolMetadata(t *testing.T) {
	request := GeneralOpenAIRequest{
		Model:    "gpt-4.1",
		Messages: []Message{{Role: "user", Content: "你好"}},
		Tools:    []ToolCallRequest{{Function: FunctionRequest{Name: "user_lookup", Description: "user metadata"}}},
	}

	meta := request.GetTokenCountMeta()
	require.NotNil(t, meta)
	require.Contains(t, meta.CombineText, "user")
	require.Contains(t, meta.CombineText, "user_lookup")
	require.Equal(t, "你好", meta.SensitiveText)
	require.False(t, strings.Contains(meta.SensitiveText, "user"))
}
