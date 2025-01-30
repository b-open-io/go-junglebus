package junglebus

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin(t *testing.T) {
	// Setup test client with localRoundTripper for validation tests
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	testClient := &http.Client{Transport: localRoundTripper{handler: mux}}

	// Test with empty username
	client, err := New(
		WithHTTPClient("test-url", testClient),
	)
	require.NoError(t, err)
	err = client.Login(context.Background(), "", "test-password")
	assert.Error(t, err)
	assert.Equal(t, "username cannot be empty", err.Error())

	// Test with empty password
	err = client.Login(context.Background(), "test-user", "")
	assert.Error(t, err)
	assert.Equal(t, "password cannot be empty", err.Error())

	// Test with nil context
	err = client.Login(nil, "test-user", "test-password")
	assert.Error(t, err)
	assert.Equal(t, "context cannot be nil", err.Error())

	// Create test server for successful case
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/user/login", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)

		var reqBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)
		assert.Equal(t, "test-user", reqBody["username"])
		assert.Equal(t, "test-password", reqBody["password"])

		// Mock response
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token": "test-token",
		})
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	// Create client with test client
	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithVersion("v1"),
	)
	require.NoError(t, err)

	// Test successful login
	err = client.Login(context.Background(), "test-user", "test-password")
	require.NoError(t, err)
	transport := client.transport
	assert.Equal(t, "test-token", transport.GetToken())

	// Test server error response
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/user/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithVersion("v1"),
	)
	require.NoError(t, err)
	err = client.Login(context.Background(), "test-user", "test-password")
	assert.Error(t, err)
}

func TestGetSubscriptionToken(t *testing.T) {
	// Setup test client with localRoundTripper for validation tests
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	testClient := &http.Client{Transport: localRoundTripper{handler: mux}}

	// Test with empty subscription ID
	client, err := New(
		WithHTTPClient("test-url", testClient),
	)
	require.NoError(t, err)
	_, err = client.GetSubscriptionToken(context.Background(), "")
	assert.Error(t, err)
	assert.Equal(t, "subscription ID cannot be empty", err.Error())

	// Test with nil context
	_, err = client.GetSubscriptionToken(nil, "test-sub")
	assert.Error(t, err)
	assert.Equal(t, "context cannot be nil", err.Error())

	// Create test server for successful case
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/user/subscription-token", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		var reqBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)
		assert.Equal(t, "test-sub", reqBody["id"])

		// Mock response
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token": "test-token",
		})
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	// Create client with test client
	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithVersion("v1"),
	)
	require.NoError(t, err)

	// Test successful subscription token fetch
	token, err := client.GetSubscriptionToken(context.Background(), "test-sub")
	require.NoError(t, err)
	assert.Equal(t, "test-token", token)

	// Test server error response
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/user/subscription-token", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithVersion("v1"),
	)
	require.NoError(t, err)
	_, err = client.GetSubscriptionToken(context.Background(), "test-sub")
	assert.Error(t, err)
}

func TestRefreshToken(t *testing.T) {
	// Setup test client with localRoundTripper for validation tests
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	testClient := &http.Client{Transport: localRoundTripper{handler: mux}}

	// Test with nil context
	client, err := New(
		WithHTTPClient("test-url", testClient),
	)
	require.NoError(t, err)
	_, err = client.RefreshToken(nil)
	assert.Error(t, err)
	assert.Equal(t, "context cannot be nil", err.Error())

	// Create test server for successful case
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/user/refresh-token", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)

		// Mock response
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token": "new-token",
		})
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	// Create client with test client
	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithVersion("v1"),
		WithToken("old-token"),
	)
	require.NoError(t, err)

	// Test successful token refresh
	token, err := client.RefreshToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "new-token", token)

	// Test server error response
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/user/refresh-token", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithVersion("v1"),
	)
	require.NoError(t, err)
	_, err = client.RefreshToken(context.Background())
	assert.Error(t, err)
}
