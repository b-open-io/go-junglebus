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

func TestGetBlockHeader(t *testing.T) {
	// Setup test client with localRoundTripper for validation tests
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	testClient := &http.Client{Transport: localRoundTripper{handler: mux}}

	// Test with empty block
	client, err := New(
		WithHTTPClient("test-url", testClient),
	)
	require.NoError(t, err)
	_, err = client.GetBlockHeader(context.Background(), "")
	require.Error(t, err)
	assert.Equal(t, "block cannot be empty", err.Error())

	// Test with nil context
	_, err = client.GetBlockHeader(context.TODO(), "test-block")
	require.Error(t, err)

	// Create test server for successful case
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/block_header/get/test-block", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "test-token", r.Header.Get("token"))

		// Mock response
		header := &models.BlockHeader{
			Hash:       "test-block-hash",
			Coin:       1,
			Height:     12345,
			Time:       1234567890,
			Nonce:      987654321,
			Version:    1,
			MerkleRoot: "test-merkle-root",
			Bits:       "1d00ffff",
			Synced:     1,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(header); err != nil {
			t.Error(err)
			return
		}
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	// Create client with test client
	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithToken("test-token"),
		WithVersion("v1"),
	)
	require.NoError(t, err)

	// Test successful block header fetch
	header, err := client.GetBlockHeader(context.Background(), "test-block")
	require.NoError(t, err)
	assert.Equal(t, "test-block-hash", header.Hash)
	assert.Equal(t, uint32(1), header.Coin)
	assert.Equal(t, uint32(12345), header.Height)
	assert.Equal(t, uint32(1234567890), header.Time)
	assert.Equal(t, uint32(987654321), header.Nonce)
	assert.Equal(t, uint32(1), header.Version)
	assert.Equal(t, "test-merkle-root", header.MerkleRoot)
	assert.Equal(t, "1d00ffff", header.Bits)
	assert.Equal(t, uint64(1), header.Synced)

	// Test server error response
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/block_header/get/test-block", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithToken("test-token"),
	)
	require.NoError(t, err)
	_, err = client.GetBlockHeader(context.Background(), "test-block")
	require.Error(t, err)
}

func TestGetBlockHeaders(t *testing.T) {
	// Setup test client with localRoundTripper for validation tests
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	testClient := &http.Client{Transport: localRoundTripper{handler: mux}}

	// Test with empty fromBlock
	client, err := New(
		WithHTTPClient("test-url", testClient),
	)
	require.NoError(t, err)
	_, err = client.GetBlockHeaders(context.Background(), "", 10)
	assert.Error(t, err)
	assert.Equal(t, "fromBlock cannot be empty", err.Error())

	// Test with nil context
	_, err = client.GetBlockHeaders(context.TODO(), "test-block", 10)
	require.Error(t, err)

	// Test with empty block
	_, err = client.GetBlockHeaders(context.Background(), "", 10)
	require.Error(t, err)

	// Create test server for successful case
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/block_header/list/test-block", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "test-token", r.Header.Get("token"))
		assert.Equal(t, "10", r.URL.Query().Get("limit"))

		// Mock response
		headers := []*models.BlockHeader{
			{
				Hash:       "test-block-hash-1",
				Coin:       1,
				Height:     12345,
				Time:       1234567890,
				Nonce:      987654321,
				Version:    1,
				MerkleRoot: "test-merkle-root-1",
				Bits:       "1d00ffff",
				Synced:     1,
			},
			{
				Hash:       "test-block-hash-2",
				Coin:       1,
				Height:     12346,
				Time:       1234567891,
				Nonce:      987654322,
				Version:    1,
				MerkleRoot: "test-merkle-root-2",
				Bits:       "1d00ffff",
				Synced:     1,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(headers); err != nil {
			t.Error(err)
			return
		}
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	// Create client with test client
	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithToken("test-token"),
		WithVersion("v1"),
	)
	require.NoError(t, err)

	// Test successful block headers fetch
	headers, err := client.GetBlockHeaders(context.Background(), "test-block", 10)
	require.NoError(t, err)
	require.Len(t, headers, 2)

	// Verify first header
	assert.Equal(t, "test-block-hash-1", headers[0].Hash)
	assert.Equal(t, uint32(1), headers[0].Coin)
	assert.Equal(t, uint32(12345), headers[0].Height)
	assert.Equal(t, uint32(1234567890), headers[0].Time)
	assert.Equal(t, uint32(987654321), headers[0].Nonce)
	assert.Equal(t, uint32(1), headers[0].Version)
	assert.Equal(t, "test-merkle-root-1", headers[0].MerkleRoot)
	assert.Equal(t, "1d00ffff", headers[0].Bits)
	assert.Equal(t, uint64(1), headers[0].Synced)

	// Verify second header
	assert.Equal(t, "test-block-hash-2", headers[1].Hash)
	assert.Equal(t, uint32(1), headers[1].Coin)
	assert.Equal(t, uint32(12346), headers[1].Height)
	assert.Equal(t, uint32(1234567891), headers[1].Time)
	assert.Equal(t, uint32(987654322), headers[1].Nonce)
	assert.Equal(t, uint32(1), headers[1].Version)
	assert.Equal(t, "test-merkle-root-2", headers[1].MerkleRoot)
	assert.Equal(t, "1d00ffff", headers[1].Bits)
	assert.Equal(t, uint64(1), headers[1].Synced)

	// Test server error response
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/block_header/list/test-block", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithToken("test-token"),
	)
	require.NoError(t, err)
	_, err = client.GetBlockHeaders(context.Background(), "test-block", 10)
	require.Error(t, err)
}

func TestGetChainTip(t *testing.T) {
	// Setup test client with localRoundTripper for validation tests
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	testClient := &http.Client{Transport: localRoundTripper{handler: mux}}

	// Test with nil context
	client, err := New(
		WithHTTPClient("test-url", testClient),
	)
	require.NoError(t, err)
	_, err = client.GetChainTip(context.TODO())
	require.Error(t, err)

	// Create test server for successful case
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/block_header/tip", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "test-token", r.Header.Get("token"))

		// Mock response
		header := &models.BlockHeader{
			Hash:       "test-tip-hash",
			Coin:       1,
			Height:     12345,
			Time:       1234567890,
			Nonce:      987654321,
			Version:    1,
			MerkleRoot: "test-merkle-root",
			Bits:       "1d00ffff",
			Synced:     1,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(header); err != nil {
			t.Error(err)
			return
		}
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	// Create client with test client
	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithToken("test-token"),
		WithVersion("v1"),
	)
	require.NoError(t, err)

	// Test successful chain tip fetch
	header, err := client.GetChainTip(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "test-tip-hash", header.Hash)
	assert.Equal(t, uint32(1), header.Coin)
	assert.Equal(t, uint32(12345), header.Height)
	assert.Equal(t, uint32(1234567890), header.Time)
	assert.Equal(t, uint32(987654321), header.Nonce)
	assert.Equal(t, uint32(1), header.Version)
	assert.Equal(t, "test-merkle-root", header.MerkleRoot)
	assert.Equal(t, "1d00ffff", header.Bits)
	assert.Equal(t, uint64(1), header.Synced)

	// Test server error response
	mux = http.NewServeMux()
	mux.HandleFunc("/v1/block_header/tip", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	testClient = &http.Client{Transport: localRoundTripper{handler: mux}}

	client, err = New(
		WithHTTPClient("test-url", testClient),
		WithToken("test-token"),
	)
	require.NoError(t, err)
	_, err = client.GetChainTip(context.Background())
	require.Error(t, err)
}
