package transports

import (
	"net/http"
	"testing"
)

func TestWithHTTPClient(t *testing.T) {
	customClient := &http.Client{}
	client := &Client{}

	WithHTTPClient("test-server", customClient)(client)

	transport, ok := client.transport.(*TransportHTTP)
	if !ok {
		t.Errorf("Expected transport to be TransportHTTP")
		return
	}

	if transport.httpClient != customClient {
		t.Errorf("Expected HTTPClient to be set to custom client")
	}
}

func TestWithToken(t *testing.T) {
	token := "test-token"
	client := &Client{}
	client.transport = NewTransportService(&TransportHTTP{})

	WithToken(token)(client)

	transport, ok := client.transport.(*TransportHTTP)
	if !ok {
		t.Errorf("Expected transport to be TransportHTTP")
		return
	}

	if transport.token != token {
		t.Errorf("Expected token to be %s, got %s", token, transport.token)
	}
}

func TestWithSSL(t *testing.T) {
	client := &Client{}
	client.transport = NewTransportService(&TransportHTTP{})

	WithSSL(true)(client)

	transport, ok := client.transport.(*TransportHTTP)
	if !ok {
		t.Errorf("Expected transport to be TransportHTTP")
		return
	}

	if !transport.useSSL {
		t.Errorf("Expected UseSSL to be true")
	}

	WithSSL(false)(client)

	if transport.useSSL {
		t.Errorf("Expected UseSSL to be false")
	}
}

func TestWithVersion(t *testing.T) {
	version := "v1.2.3"
	client := &Client{}
	client.transport = NewTransportService(&TransportHTTP{})

	WithVersion(version)(client)

	transport, ok := client.transport.(*TransportHTTP)
	if !ok {
		t.Errorf("Expected transport to be TransportHTTP")
		return
	}

	if transport.version != version {
		t.Errorf("Expected version to be %s, got %s", version, transport.version)
	}
}
