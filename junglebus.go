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
	transportOptions []ClientOps
	subscription     *Subscription
	debug            bool
}

func (jb *Client) setDefaultOptions() {
	jb.transport, _ = transports.NewTransport(
		transports.WithHTTP(DefaultServer),
	)
}

// New create a new jungle bus client
func New(opts ...ClientOps) (*Client, error) {
	client := &Client{
		transportOptions: make([]ClientOps, 0),
	}

	// If no options provided, use defaults
	if len(opts) == 0 {
		client.setDefaultOptions()
		return client, nil
	}

	// Apply all client options and store them for potential reconnection
	for _, opt := range opts {
		client.transportOptions = append(client.transportOptions, opt)
		opt(client)
	}

	return client, nil
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
