package transports

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GorillaPool/go-junglebus/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransportHTTP_SetDebug(t *testing.T) {
	transport := &TransportHTTP{}
	assert.False(t, transport.IsDebug())

	transport.SetDebug(true)
	assert.True(t, transport.IsDebug())

	transport.SetDebug(false)
	assert.False(t, transport.IsDebug())
}

func TestTransportHTTP_SSL(t *testing.T) {
	transport := &TransportHTTP{}
	assert.False(t, transport.IsSSL())

	transport.UseSSL(true)
	assert.True(t, transport.IsSSL())

	transport.UseSSL(false)
	assert.False(t, transport.IsSSL())
}

func TestTransportHTTP_Token(t *testing.T) {
	transport := &TransportHTTP{}
	assert.Empty(t, transport.GetToken())

	transport.SetToken("test-token")
	assert.Equal(t, "test-token", transport.GetToken())
}

func TestTransportHTTP_Version(t *testing.T) {
	transport := &TransportHTTP{}
	transport.SetVersion("v2")
	assert.Equal(t, "v2", transport.version)
}

func TestTransportHTTP_GetServerURL(t *testing.T) {
	transport := &TransportHTTP{server: "test.com"}
	assert.Equal(t, "test.com", transport.GetServerURL())
}

func TestTransportHTTP_GetSubscriptionToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/user/subscription-token", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var reqBody map[string]interface{}
		var decodeErr error
		if decodeErr = json.NewDecoder(r.Body).Decode(&reqBody); decodeErr != nil {
			t.Error(decodeErr)
			return
		}
		assert.Equal(t, "test-sub", reqBody[FieldSubscriptionID])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LoginResponse{Token: "test-token"})
	}))
	defer ts.Close()

	transport := &TransportHTTP{
		server:     ts.Listener.Addr().String(),
		httpClient: http.DefaultClient,
		version:    "v1",
	}

	token, err := transport.GetSubscriptionToken(context.Background(), "test-sub")
	require.NoError(t, err)
	assert.Equal(t, "test-token", token)
}

func TestTransportHTTP_RefreshToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/user/refresh-token", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LoginResponse{Token: "new-token"})
	}))
	defer ts.Close()

	transport := &TransportHTTP{
		server:     ts.Listener.Addr().String(),
		httpClient: http.DefaultClient,
		version:    "v1",
	}

	token, err := transport.RefreshToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "new-token", token)
}

func TestTransportHTTP_Login(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/user/login", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		var reqBody map[string]interface{}
		var decodeErr error
		if decodeErr = json.NewDecoder(r.Body).Decode(&reqBody); decodeErr != nil {
			t.Error(decodeErr)
			return
		}
		assert.Equal(t, "testuser", reqBody[FieldUsername])
		assert.Equal(t, "testpass", reqBody[FieldPassword])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token": "login-token",
		})
	}))
	defer ts.Close()

	transport := &TransportHTTP{
		server:     ts.Listener.Addr().String(),
		httpClient: http.DefaultClient,
		version:    "v1",
	}

	err := transport.Login(context.Background(), "testuser", "testpass")
	require.NoError(t, err)
	assert.Equal(t, "login-token", transport.GetToken())
}

func TestTransportHTTP_GetTransaction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/transaction/get/test-tx", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&models.Transaction{
			ID: "test-tx",
		})
	}))
	defer ts.Close()

	transport := &TransportHTTP{
		server:     ts.Listener.Addr().String(),
		httpClient: http.DefaultClient,
		version:    "v1",
	}

	tx, err := transport.GetTransaction(context.Background(), "test-tx")
	require.NoError(t, err)
	assert.Equal(t, "test-tx", tx.ID)
}

func TestTransportHTTP_GetAddressTransactions(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/address/get/test-addr", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]*models.Address{
			{Address: "test-addr"},
		})
	}))
	defer ts.Close()

	transport := &TransportHTTP{
		server:     ts.Listener.Addr().String(),
		httpClient: http.DefaultClient,
		version:    "v1",
	}

	addrs, err := transport.GetAddressTransactions(context.Background(), "test-addr")
	require.NoError(t, err)
	assert.Len(t, addrs, 1)
	assert.Equal(t, "test-addr", addrs[0].Address)
}

func TestTransportHTTP_GetAddressTransactionDetails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/address/transactions/test-addr", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]*models.Transaction{
			{ID: "test-tx"},
		})
	}))
	defer ts.Close()

	transport := &TransportHTTP{
		server:     ts.Listener.Addr().String(),
		httpClient: http.DefaultClient,
		version:    "v1",
	}

	txs, err := transport.GetAddressTransactionDetails(context.Background(), "test-addr")
	require.NoError(t, err)
	assert.Len(t, txs, 1)
	assert.Equal(t, "test-tx", txs[0].ID)
}

func TestTransportHTTP_GetBlockHeader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/block_header/get/123", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&models.BlockHeader{
			Height: 123,
		})
	}))
	defer ts.Close()

	transport := &TransportHTTP{
		server:     ts.Listener.Addr().String(),
		httpClient: http.DefaultClient,
		version:    "v1",
	}

	header, err := transport.GetBlockHeader(context.Background(), "123")
	require.NoError(t, err)
	assert.Equal(t, uint32(123), header.Height)
}

func TestTransportHTTP_GetBlockHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/block_header/list/123", r.URL.Path)
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]*models.BlockHeader{
			{Height: 123},
			{Height: 124},
		})
	}))
	defer ts.Close()

	transport := &TransportHTTP{
		server:     ts.Listener.Addr().String(),
		httpClient: http.DefaultClient,
		version:    "v1",
	}

	headers, err := transport.GetBlockHeaders(context.Background(), "123", 10)
	require.NoError(t, err)
	assert.Len(t, headers, 2)
	assert.Equal(t, uint32(123), headers[0].Height)
	assert.Equal(t, uint32(124), headers[1].Height)
}

func TestTransportHTTP_ErrorHandling(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	transport := &TransportHTTP{
		server:     ts.Listener.Addr().String(),
		httpClient: http.DefaultClient,
		version:    "v1",
	}

	_, err := transport.GetTransaction(context.Background(), "test-tx")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestTransportHTTP_InvalidURL(t *testing.T) {
	transport := &TransportHTTP{
		server:     "invalid-url",
		httpClient: http.DefaultClient,
		version:    "v1",
	}

	_, err := transport.GetTransaction(context.Background(), "test-tx")
	require.Error(t, err)
}

func TestTransportHTTP_GetVersion(t *testing.T) {
	transport := &TransportHTTP{
		version: "v1.2.3",
	}

	version := transport.GetVersion()

	if version != "v1.2.3" {
		t.Errorf("Expected version to be v1.2.3, got %s", version)
	}
}
