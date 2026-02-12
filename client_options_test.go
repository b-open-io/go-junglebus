package junglebus

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithHTTP(t *testing.T) {
	client := &Client{}
	opt := WithHTTP("test-url")
	opt(client)
	assert.NotNil(t, client.transport)
	assert.Equal(t, "test-url", client.transport.GetServerURL())
}

func TestWithHTTPClient(t *testing.T) {
	client := &Client{}
	httpClient := &http.Client{}
	// First initialize transport
	WithHTTP("test-url")(client)
	assert.NotNil(t, client.transport)

	// Then test custom HTTP client
	opt := WithHTTPClient("test-url", httpClient)
	opt(client)
	assert.NotNil(t, client.transport)
	assert.Equal(t, "test-url", client.transport.GetServerURL())
}

func TestWithToken(t *testing.T) {
	client := &Client{}
	// First initialize transport
	WithHTTP("test-url")(client)
	assert.NotNil(t, client.transport)

	// Then test token
	opt := WithToken("test-token")
	opt(client)
	assert.Equal(t, "test-token", client.transport.GetToken())
}

func TestWithDebugging(t *testing.T) {
	client := &Client{}
	// First initialize transport
	WithHTTP("test-url")(client)
	assert.NotNil(t, client.transport)

	// Then test debug
	opt := WithDebugging(true)
	opt(client)
	assert.True(t, client.debug)
	assert.True(t, client.transport.IsDebug())
}

func TestWithSSL(t *testing.T) {
	client := &Client{}
	// First initialize transport
	WithHTTP("test-url")(client)
	assert.NotNil(t, client.transport)

	// Then test SSL
	opt := WithSSL(true)
	opt(client)
	assert.True(t, client.transport.IsSSL())
}

func TestWithVersion(t *testing.T) {
	client := &Client{}
	// First initialize transport
	WithHTTP("test-url")(client)
	assert.NotNil(t, client.transport)

	// Then test version
	opt := WithVersion("v2")
	opt(client)
	assert.Equal(t, "v2", client.transport.GetVersion())
}

func TestClientOptions_NilClient(t *testing.T) {
	// Test that options don't panic with nil client
	WithHTTP("test.com")(nil)
	WithHTTPClient("test.com", nil)(nil)
	WithToken("test")(nil)
	WithDebugging(true)(nil)
	WithSSL(true)(nil)
	WithVersion("v2")(nil)
}

func TestClientOptions_DefaultValues(t *testing.T) {
	client, err := New()
	require.NoError(t, err)

	transport := client.GetTransport()
	assert.NotNil(t, transport)
	assert.False(t, client.debug)
	assert.True(t, (*transport).IsSSL())
	assert.Equal(t, DefaultServer, (*transport).GetServerURL())
}

func TestClientOptions_Chaining(t *testing.T) {
	client := &Client{}
	WithHTTP("test-url")(client)
	WithToken("test-token")(client)
	WithSSL(true)(client)
	WithVersion("v2")(client)
	assert.NotNil(t, client.transport)
	assert.Equal(t, "test-url", client.transport.GetServerURL())
	assert.Equal(t, "test-token", client.transport.GetToken())
	assert.True(t, client.transport.IsSSL())
	assert.Equal(t, "v2", client.transport.GetVersion())
}
