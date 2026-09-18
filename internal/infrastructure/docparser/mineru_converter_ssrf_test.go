package docparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMinerUOutboundURL_RejectsLoopback(t *testing.T) {
	withDocparserSSRFWhitelist(t, "")
	err := validateMinerUOutboundURL("http://127.0.0.1:8080")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SSRF")
}

func TestPingMinerU_RejectsPrivateEndpoint(t *testing.T) {
	withDocparserSSRFWhitelist(t, "")
	ok, msg := PingMinerU("http://127.0.0.1:8080")
	assert.False(t, ok)
	assert.Contains(t, msg, "SSRF")
}
