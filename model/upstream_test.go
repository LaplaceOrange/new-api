package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeUpstreamURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "trims default https port and slash", raw: " HTTPS://Example.COM:443/v1/ ", want: "https://example.com/v1"},
		{name: "keeps non-default port", raw: "http://Example.COM:8080/api/", want: "http://example.com:8080/api"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeUpstreamURL(tc.raw)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestNormalizeUpstreamURLRejectsCredentialsAndQuery(t *testing.T) {
	for _, raw := range []string{"https://user:pass@example.com", "https://example.com?token=secret", "file:///tmp/x"} {
		_, err := NormalizeUpstreamURL(raw)
		require.Error(t, err, raw)
	}
}
