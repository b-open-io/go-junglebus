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

// getNilContext is used for testing nil context cases
//
//nolint:staticcheck // Intentionally returns nil for testing
func getNilContext() context.Context {
	return nil
}

func TestGetAddressTransactions(t *testing.T) {
	// Setup test client with localRoundTripper for validation tests
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	testClient := &http.Client{Transport: localRoundTripper{handler: mux}}

	// Test with empty address
	client, err := New(
		WithHTTPClient("test-url", testClient),
	)
	require.NoError(t, err)
	_, err = client.GetAddressTransactions(context.Background(), "")
	require.Error(t, err)
	assert.Equal(t, "address cannot be empty", err.Error())

	// Test with nil context
	_, err = client.GetAddressTransactions(getNilContext(), "test-address")
	require.Error(t, err)
	assert.Equal(t, "context cannot be nil", err.Error())

	// Create test server for successful case
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/address/get/test-address", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "test-token", r.Header.Get("token"))

		// Mock response
		addresses := []*models.Address{
			{
				ID:            "test-id",
				Address:       "test-address",
				TransactionID: "test-tx-id",
				BlockHash:     "test-block-hash",
				BlockIndex:    1,
			},
		}
		json.NewEncoder(w).Encode(addresses)
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	// Create client with test client
	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithToken("test-token"),
		WithVersion("v1"),
	)
	require.NoError(t, err)

	// Test successful address fetch
	addresses, err := client.GetAddressTransactions(context.Background(), "test-address")
	require.NoError(t, err)
	require.Len(t, addresses, 1)
	assert.Equal(t, "test-id", addresses[0].ID)
	assert.Equal(t, "test-address", addresses[0].Address)
	assert.Equal(t, "test-tx-id", addresses[0].TransactionID)
	assert.Equal(t, "test-block-hash", addresses[0].BlockHash)
	assert.Equal(t, uint64(1), addresses[0].BlockIndex)

	// Test server error response
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/address/get/test-address", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithToken("test-token"),
	)
	require.NoError(t, err)
	_, err = client.GetAddressTransactions(context.Background(), "test-address")
	require.Error(t, err)
}

func TestGetAddressTransactionDetails(t *testing.T) {
	// Setup test client with localRoundTripper for validation tests
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	testClient := &http.Client{Transport: localRoundTripper{handler: mux}}

	// Test with empty address
	client, err := New(
		WithHTTPClient("test-url", testClient),
	)
	require.NoError(t, err)
	_, err = client.GetAddressTransactionDetails(context.Background(), "")
	require.Error(t, err)
	assert.Equal(t, "address cannot be empty", err.Error())

	// Test with nil context
	_, err = client.GetAddressTransactionDetails(getNilContext(), "test-address")
	require.Error(t, err)
	assert.Equal(t, "context cannot be nil", err.Error())

	// Create test server for successful case
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/address/transactions/test-address", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "test-token", r.Header.Get("token"))

		// Mock response
		transactions := []*models.Transaction{
			{
				ID:          "test-tx-1",
				BlockHash:   "test-block-hash-1",
				BlockHeight: 12345,
				BlockIndex:  1,
				BlockTime:   1234567890,
				Transaction: []byte("test-tx-data-1"),
				MerkleProof: []byte("test-merkle-data-1"),
			},
			{
				ID:          "test-tx-2",
				BlockHash:   "test-block-hash-2",
				BlockHeight: 12346,
				BlockIndex:  2,
				BlockTime:   1234567891,
				Transaction: []byte("test-tx-data-2"),
				MerkleProof: []byte("test-merkle-data-2"),
			},
		}
		json.NewEncoder(w).Encode(transactions)
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	// Create client with test client
	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithToken("test-token"),
		WithVersion("v1"),
	)
	require.NoError(t, err)

	// Test successful transaction details fetch
	transactions, err := client.GetAddressTransactionDetails(context.Background(), "test-address")
	require.NoError(t, err)
	require.Len(t, transactions, 2)

	// Verify first transaction
	assert.Equal(t, "test-tx-1", transactions[0].ID)
	assert.Equal(t, "test-block-hash-1", transactions[0].BlockHash)
	assert.Equal(t, uint32(12345), transactions[0].BlockHeight)
	assert.Equal(t, uint64(1), transactions[0].BlockIndex)
	assert.Equal(t, uint32(1234567890), transactions[0].BlockTime)
	assert.Equal(t, []byte("test-tx-data-1"), transactions[0].Transaction)
	assert.Equal(t, []byte("test-merkle-data-1"), transactions[0].MerkleProof)

	// Verify second transaction
	assert.Equal(t, "test-tx-2", transactions[1].ID)
	assert.Equal(t, "test-block-hash-2", transactions[1].BlockHash)
	assert.Equal(t, uint32(12346), transactions[1].BlockHeight)
	assert.Equal(t, uint64(2), transactions[1].BlockIndex)
	assert.Equal(t, uint32(1234567891), transactions[1].BlockTime)
	assert.Equal(t, []byte("test-tx-data-2"), transactions[1].Transaction)
	assert.Equal(t, []byte("test-merkle-data-2"), transactions[1].MerkleProof)

	// Test server error response
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/address/transactions/test-address", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithToken("test-token"),
	)
	require.NoError(t, err)
	_, err = client.GetAddressTransactionDetails(context.Background(), "test-address")
	require.Error(t, err)
}
