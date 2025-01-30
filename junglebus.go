// Package junglebus is a Go client for interacting with GorillaPool's JungleBus
//
// If you have any suggestions or comments, please feel free to open an issue on
// this GitHub repository!
//
// By GorillaPool (https://githujb.com/b-open-io)
package junglebus

import (
	"github.com/b-open-io/go-junglebus/transports"
)

var DefaultServer = "junglebus.gorillapool.io"

// ClientOps are used for client options
type ClientOps func(c *Client)

// Client is the go-junglebus client
type Client struct {
	transports.TransportService
	transport        transports.TransportService
	transportOptions []transports.ClientOps
	subscription     *Subscription
	debug            bool
}

// New create a new jungle bus client
func New(opts ...ClientOps) (*Client, error) {
	client := &Client{
		transportOptions: make([]transports.ClientOps, 0),
	}

	client.setDefaultOptions()

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

func (jb *Client) setDefaultOptions() {
	defaultOpt := transports.WithHTTP(DefaultServer)
	jb.transportOptions = []transports.ClientOps{defaultOpt}
	jb.transport, _ = transports.NewTransport(defaultOpt)
}

// SetDebug turn the debugging on or off
func (jb *Client) SetDebug(debug bool) {
	jb.debug = debug
}

// IsDebug return the debugging status
func (jb *Client) IsDebug() bool {
	return jb.debug
}

// GetTransport returns the current transport service
func (jb *Client) GetTransport() *transports.TransportService {
	return &jb.transport
}
