package junglebus

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/b-open-io/go-junglebus/transports"
	"github.com/stretchr/testify/require"
)

//nolint:unused // These constants and helpers are kept for future test cases
const (
	requestTypeHTTP = "http"
	serverURL       = "junglebus.gorillapool.io" // Removed https:// prefix
	transactionJSON = `{"id":"041479f86c475603fd510431cf702bc8c9849a9c350390eb86b467d82a13cc24","created_at":"2022-01-28T13:45:01.711Z","updated_at":null,"deleted_at":null,"hex":"0100000004afcafa163824904aa3bbc403b30db56a08f29ffa53b16b1b4b4914b9bd7d7610010000006a4730440220710c2b2fe5a0ece2cbc962635d0fb6dabf95c94db0b125c3e2613cede9738666022067e9cc0f4f706c3a2781990981a50313fb0aad18c1e19a757125eec2408ecadb412103dcd8d28545c9f80af54648fcca87972d89e3e7ed7b482465dd78b62c784ad533ffffffff783452c4038c46a4d68145d829f09c70755edd8d4b3512d7d6a27db08a92a76b000000006b483045022100ee7e24859274013e748090a022bf51200ab216771b5d0d57c0d074843dfa62bd02203933c2bd2880c2f8257befff44dc19cb1f3760c6eea44fc0f8094ff94bce652a41210375680e36c45658bd9b0694a48f5756298cf95b77f50bada14ef1cba6d7ea1d3affffffff25e893beb8240ede7661c02cb959799d364711ba638eccdf12e3ce60faa2fd0f010000006b483045022100fc380099ac7f41329aaeed364b95baa390be616243b80a8ef444ae0ddc76fa3a0220644a9677d40281827fa4602269720a5a453fbe77409be40293c3f8248534e5f8412102398146eff37de36ed608b2ee917a3d4b4a424722f9a00f1b48c183322a8ef2a1ffffffff00e6f915a5a3678f01229e5c320c64755f242be6cebfac54e2f77ec5e0eec581000000006b483045022100951511f81291ac234926c866f777fe8e77bc00661031675978ddecf159cc265902207a5957dac7c89493e2b7df28741ce3291e19dc8bba4b13082c69d0f2b79c70ab4121031d674b3ad42b28f3a445e9970bd9ae8fe5d3fb89ee32452d9f6dc7916ea184bfffffffff04c7110000000000001976a91483615db3fb9b9cbbf4cd407100833511a1cb278588ac30060000000000001976a914296a5295e70697e844fb4c2113b41a501d41452e88ac96040000000000001976a914e73e21935fc48df0d1cf8b73f2e8bbd23b78244a88ac27020000000000001976a9140b2b03751813e3467a28ce916cbb102d84c6eec588ac00000000","block_hash":"","block_height":0,"fee":354,"number_of_inputs":4,"number_of_outputs":4,"total_value":6955,"metadata":{"client_id":"8","run":76,"run_id":"3108aa426fc7102488bb0ffd","xbench":"is awesome"},"output_value":1725,"direction":"incoming"}`
	txID            = "041479f86c475603fd510431cf702bc8c9849a9c350390eb86b467d82a13cc24"
)

// localRoundTripper is a http.RoundTripper that executes HTTP transactions
// by using handler directly, instead of going over an HTTP connection.
type localRoundTripper struct {
	handler http.Handler
}

func (l localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	l.handler.ServeHTTP(w, req)
	return w.Result(), nil
}

//nolint:unused // Helper function for writing test data
func mustWrite(w io.Writer, s string) {
	_, err := io.WriteString(w, s)
	if err != nil {
		panic(err)
	}
}

//nolint:unused // Test transport handler types for future test cases
type testTransportHandler struct {
	ClientURL string
	Client    func(serverURL string, httpClient *http.Client) ClientOps
	Path      string
	Queries   []*testTransportHandlerRequest
	Result    string
	Type      string
}

//nolint:unused // Test transport handler request type for future test cases
type testTransportHandlerRequest struct {
	Path   string
	Result func(w http.ResponseWriter, req *http.Request)
}

// TestNewJungleBusClient will test the JungleBusClient method
func TestNewJungleBusClient(t *testing.T) {
	t.Run("new client - no options", func(t *testing.T) {
		client, err := New()
		require.NoError(t, err)
		require.IsType(t, Client{}, *client)
	})
}

// TestGetTransport will test the GetTransport method
func TestGetTransport(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	testClient := &http.Client{Transport: localRoundTripper{handler: mux}}

	t.Run("default transport", func(t *testing.T) {
		client, _ := New(
			WithHTTPClient("test.com", testClient),
			WithSSL(true),
		)
		transport := client.GetTransport()
		require.IsType(t, &transports.TransportHTTP{}, *transport)
		require.False(t, (*transport).IsDebug())
		require.True(t, (*transport).IsSSL())
	})

	t.Run("transport with debug and no SSL", func(t *testing.T) {
		client, _ := New(
			WithHTTPClient("test.com", testClient),
			WithDebugging(true),
			WithSSL(false),
		)
		transport := client.GetTransport()
		require.IsType(t, &transports.TransportHTTP{}, *transport)
		require.True(t, (*transport).IsDebug())
		require.False(t, (*transport).IsSSL())
	})

	t.Run("transport with explicit protocol", func(t *testing.T) {
		client, _ := New(
			WithHTTPClient("http://test.com", testClient),
			WithSSL(false),
		)
		transport := client.GetTransport()
		require.IsType(t, &transports.TransportHTTP{}, *transport)
		require.False(t, (*transport).IsDebug())
		require.False(t, (*transport).IsSSL())
	})
}

//nolint:unused // Helper function for creating test clients
func getTestClient(transportHandler testTransportHandler) *Client {
	mux := http.NewServeMux()
	if transportHandler.Queries != nil {
		for _, query := range transportHandler.Queries {
			mux.HandleFunc(query.Path, query.Result)
		}
	} else {
		mux.HandleFunc(transportHandler.Path, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			mustWrite(w, transportHandler.Result)
		})
	}
	httpclient := &http.Client{Transport: localRoundTripper{handler: mux}}

	opts := []ClientOps{
		transportHandler.Client(transportHandler.ClientURL, httpclient),
	}

	client, _ := New(opts...)

	return client
}

func TestTransportHandlerQueries(t *testing.T) {
	handler := testTransportHandler{
		ClientURL: "test.com",
		Client: func(serverURL string, httpClient *http.Client) ClientOps {
			return WithHTTPClient(serverURL, httpClient)
		},
		Type: requestTypeHTTP,
		Queries: []*testTransportHandlerRequest{
			{
				Path: "/v1/transaction/get/" + txID,
				Result: func(w http.ResponseWriter, req *http.Request) {
					require.Equal(t, "GET", req.Method)
					w.Header().Set("Content-Type", "application/json")
					mustWrite(w, transactionJSON)
				},
			},
		},
	}

	client := getTestClient(handler)
	tx, err := client.GetTransaction(context.Background(), txID)
	require.NoError(t, err)
	require.Equal(t, txID, tx.ID)
}
