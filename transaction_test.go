package junglebus

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/b-open-io/go-junglebus/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTransaction(t *testing.T) {
	// Setup test client with localRoundTripper for validation tests
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	testClient := &http.Client{Transport: localRoundTripper{handler: mux}}

	// Test with empty transaction ID first
	client, err := New(
		WithHTTPClient("test-url", testClient),
	)
	require.NoError(t, err)
	_, err = client.GetTransaction(context.Background(), "")
	require.Error(t, err)
	assert.Equal(t, "transaction ID cannot be empty", err.Error())

	// Test with nil context
	_, err = client.GetTransaction(context.TODO(), "test-tx-id")
	require.Error(t, err)

	// Create test server for successful case
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/transaction/get/test-tx-id", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "test-token", r.Header.Get("token"))

		// Mock response
		tx := &models.Transaction{
			ID:          "test-tx-id",
			BlockHash:   "test-block-hash",
			BlockHeight: 12345,
			BlockIndex:  1,
			BlockTime:   1234567890,
			Transaction: []byte("test-tx-data"),
			MerkleProof: []byte("test-merkle-data"),
		}
		json.NewEncoder(w).Encode(tx)
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	// Create client with test client
	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithToken("test-token"),
		WithVersion("v1"),
	)
	require.NoError(t, err)

	// Test successful transaction fetch
	tx, err := client.GetTransaction(context.Background(), "test-tx-id")
	require.NoError(t, err)
	assert.Equal(t, "test-tx-id", tx.ID)
	assert.Equal(t, "test-block-hash", tx.BlockHash)
	assert.Equal(t, uint32(12345), tx.BlockHeight)
	assert.Equal(t, uint64(1), tx.BlockIndex)
	assert.Equal(t, uint32(1234567890), tx.BlockTime)
	assert.Equal(t, []byte("test-tx-data"), tx.Transaction)
	assert.Equal(t, []byte("test-merkle-data"), tx.MerkleProof)

	// Test server error response
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/transaction/get/test-tx-id", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithToken("test-token"),
	)
	require.NoError(t, err)
	_, err = client.GetTransaction(context.Background(), "test-tx-id")
	require.Error(t, err)
}

func TestGetTransaction_WithDebug(t *testing.T) {
	// Create test handler
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transaction/get/test-tx-id", func(w http.ResponseWriter, _ *http.Request) {
		tx := &models.Transaction{ID: "test-tx-id"}
		json.NewEncoder(w).Encode(tx)
	})
	testClient := &http.Client{Transport: localRoundTripper{handler: mux}}

	// Create client with debug enabled
	client, err := New(
		WithHTTPClient("test-url", testClient),
		WithToken("test-token"),
		WithDebugging(true),
	)
	require.NoError(t, err)

	// Test transaction fetch with debug enabled
	tx, err := client.GetTransaction(context.Background(), "test-tx-id")
	require.NoError(t, err)
	assert.Equal(t, "test-tx-id", tx.ID)
}
