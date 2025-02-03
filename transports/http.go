package transports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/b-open-io/go-junglebus/models"
)

// TransportHTTP is the struct for HTTP
type TransportHTTP struct {
	debug      bool
	httpClient *http.Client
	server     string
	token      string
	useSSL     bool
	version    string
}

// SetDebug turn the debugging on or off
func (h *TransportHTTP) SetDebug(debug bool) {
	h.debug = debug
}

// IsDebug return the debugging status
func (h *TransportHTTP) IsDebug() bool {
	return h.debug
}

// UseSSL turn the SSL on or off
func (h *TransportHTTP) UseSSL(useSSL bool) {
	h.useSSL = useSSL
}

// IsSSL return the SSL status
func (h *TransportHTTP) IsSSL() bool {
	return h.useSSL
}

// SetToken sets the token to use for all requests manually
func (h *TransportHTTP) SetToken(token string) {
	h.token = token
}

// GetToken gets the token to use for all requests
func (h *TransportHTTP) GetToken() string {
	return h.token
}

// GetSubscriptionToken gets a token based on the subscription ID
func (h *TransportHTTP) GetSubscriptionToken(ctx context.Context, subscriptionID string) (string, error) {

	jsonStr, err := json.Marshal(map[string]interface{}{
		FieldSubscriptionID: subscriptionID,
	})
	if err != nil {
		return "", err
	}

	var response LoginResponse
	if err = h.doHTTPRequest(
		ctx, http.MethodPost, `/user/subscription-token`, jsonStr, &response,
	); err != nil {
		return "", err
	}

	return response.Token, nil
}

// RefreshToken gets a new  token to use for all requests
func (h *TransportHTTP) RefreshToken(ctx context.Context) (string, error) {
	var response LoginResponse
	if err := h.doHTTPRequest(
		ctx, http.MethodGet, `/user/refresh-token`, nil, &response,
	); err != nil {
		return "", err
	}

	return response.Token, nil
}

// SetVersion sets the version to use for all calls
func (h *TransportHTTP) SetVersion(version string) {
	h.version = version
}

// GetVersion gets the version used for all calls
func (h *TransportHTTP) GetVersion() string {
	return h.version
}

// GetServerURL get the server URL for this transport
func (h *TransportHTTP) GetServerURL() string {
	return h.server
}

func (h *TransportHTTP) Login(ctx context.Context, username string, password string) error {

	jsonStr, err := json.Marshal(map[string]interface{}{
		FieldUsername: username,
		FieldPassword: password,
	})
	if err != nil {
		return err
	}

	var loginResponse map[string]interface{}
	if err = h.doHTTPRequest(
		ctx, http.MethodGet, `/user/login`, jsonStr, &loginResponse,
	); err != nil {
		return err
	}
	if h.debug {
		log.Printf("Login: %v\n", loginResponse)
	}

	if token, ok := loginResponse["token"]; ok {
		h.SetToken(token.(string))
		return nil
	}

	return ErrFailedLogin
}

// GetTransaction will get a transaction by ID
func (h *TransportHTTP) GetTransaction(ctx context.Context, txID string) (transaction *models.Transaction, err error) {

	if err = h.doHTTPRequest(
		ctx, http.MethodGet, "/transaction/get/"+txID, nil, &transaction,
	); err != nil {
		return nil, err
	}
	if h.debug {
		log.Printf("Transaction: %v\n", transaction)
	}

	return transaction, nil
}

// GetAddressTransactions will get the metadata of all transaction related to the given address
func (h *TransportHTTP) GetAddressTransactions(ctx context.Context, address string, fromHeight uint32) (addr []*models.AddressTx, err error) {
	url := fmt.Sprintf("/address/get/%s/%d", address, fromHeight)
	if err = h.doHTTPRequest(ctx, http.MethodGet, url, nil, &addr); err != nil {
		return nil, err
	}
	if h.debug {
		log.Printf("Address transactions: %v\n", addr)
	}
	return addr, nil
}

// GetAddressTransactionDetails will get all transactions related to the given address
func (h *TransportHTTP) GetAddressTransactionDetails(ctx context.Context, address string, fromHeight uint32) (transactions []*models.Transaction, err error) {
	url := fmt.Sprintf("/address/transactions/%s/%d", address, fromHeight)
	if err = h.doHTTPRequest(ctx, http.MethodGet, url, nil, &transactions); err != nil {
		return nil, err
	}
	if h.debug {
		log.Printf("transactions: %d\n", len(transactions))
	}
	return transactions, nil
}

// GetBlockHeader will get the given block header details
// Can pass either the block hash or the block height (as a string)
func (h *TransportHTTP) GetBlockHeader(ctx context.Context, block string) (blockHeader *models.BlockHeader, err error) {

	if err = h.doHTTPRequest(
		ctx, http.MethodGet, "/block_header/get/"+block, nil, &blockHeader,
	); err != nil {
		return nil, err
	}
	if h.debug {
		log.Printf("transactions: %v\n", blockHeader)
	}

	return blockHeader, nil
}

// GetBlockHeaders will get all block headers from the given block, limited by limit
// Can pass either the block hash or the block height (as a string)
func (h *TransportHTTP) GetBlockHeaders(ctx context.Context, fromBlock string, limit uint) (blockHeaders []*models.BlockHeader, err error) {

	if err = h.doHTTPRequest(
		ctx, http.MethodGet, fmt.Sprintf("/block_header/list/%s?limit=%d", fromBlock, limit), nil, &blockHeaders,
	); err != nil {
		return nil, err
	}
	if h.debug {
		log.Printf("transactions: %v\n", blockHeaders)
	}

	return blockHeaders, nil
}

/* Missing methods added to fully implement TransportService */

func (h *TransportHTTP) GetRawTransaction(ctx context.Context, txID string) ([]byte, error) {
	url := "/transaction/raw/" + txID
	var raw []byte
	if err := h.doHTTPRequest(ctx, http.MethodGet, url, nil, &raw); err != nil {
		return nil, err
	}
	if h.debug {
		log.Printf("Raw transaction: %v\n", raw)
	}
	return raw, nil
}

func (h *TransportHTTP) GetFromBlock(ctx context.Context, subscriptionID string, height uint32, lastIdx uint64) ([]*models.Transaction, error) {
	url := fmt.Sprintf("/transaction/from_block/%s?height=%d&last_idx=%d", subscriptionID, height, lastIdx)
	var transactions []*models.Transaction
	if err := h.doHTTPRequest(ctx, http.MethodGet, url, nil, &transactions); err != nil {
		return nil, err
	}
	if h.debug {
		log.Printf("GetFromBlock transactions: %d\n", len(transactions))
	}
	return transactions, nil
}

func (h *TransportHTTP) GetLiteFromBlock(ctx context.Context, subscriptionID string, height uint32, lastIdx uint64) ([]*models.TransactionResponse, error) {
	url := fmt.Sprintf("/transaction/from_block/lite/%s?height=%d&last_idx=%d", subscriptionID, height, lastIdx)
	var transactions []*models.TransactionResponse
	if err := h.doHTTPRequest(ctx, http.MethodGet, url, nil, &transactions); err != nil {
		return nil, err
	}
	if h.debug {
		log.Printf("GetLiteFromBlock transactions: %d\n", len(transactions))
	}
	return transactions, nil
}

func (h *TransportHTTP) GetTxo(ctx context.Context, txID string, vout uint32) ([]byte, error) {
	url := fmt.Sprintf("/txo/get/%s/%d", txID, vout)
	var data []byte
	if err := h.doHTTPRequest(ctx, http.MethodGet, url, nil, &data); err != nil {
		return nil, err
	}
	if h.debug {
		log.Printf("GetTxo: %v\n", data)
	}
	return data, nil
}

func (h *TransportHTTP) GetSpend(ctx context.Context, txID string, vout uint32) ([]byte, error) {
	url := fmt.Sprintf("/txo/spend/%s/%d", txID, vout)
	var data []byte
	if err := h.doHTTPRequest(ctx, http.MethodGet, url, nil, &data); err != nil {
		return nil, err
	}
	if h.debug {
		log.Printf("GetSpend: %v\n", data)
	}
	return data, nil
}

func (h *TransportHTTP) GetUser(ctx context.Context) (*models.User, error) {
	url := "/user/get"
	var user *models.User
	if err := h.doHTTPRequest(ctx, http.MethodGet, url, nil, &user); err != nil {
		return nil, err
	}
	if h.debug {
		log.Printf("GetUser: %v\n", user)
	}
	return user, nil
}

// GetChainTip will get the current chain tip block header
func (h *TransportHTTP) GetChainTip(ctx context.Context) (blockHeader *models.BlockHeader, err error) {
	if err = h.doHTTPRequest(
		ctx, http.MethodGet, "/block_header/tip", nil, &blockHeader,
	); err != nil {
		return nil, err
	}
	if h.debug {
		log.Printf("chain tip: %v\n", blockHeader)
	}
	return blockHeader, nil
}

// doHTTPRequest will create and submit the HTTP request
func (h *TransportHTTP) doHTTPRequest(ctx context.Context, method string, path string, rawJSON []byte, responseJSON interface{}) error {
	protocol := "https"
	if !h.useSSL {
		protocol = "http"
	}
	serverRequest := fmt.Sprintf("%s://%s/%s%s", protocol, h.server, h.version, path)
	req, err := http.NewRequestWithContext(ctx, method, serverRequest, bytes.NewBuffer(rawJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("token", h.token)

	var resp *http.Response
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	if resp, err = h.httpClient.Do(req); err != nil {
		return err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return errors.New("server error: " + strconv.Itoa(resp.StatusCode) + " - " + resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(responseJSON)
}
