package junglebus

import (
	"net/http"

	"github.com/GorillaPool/go-junglebus/transports"
)

// WithHTTP will overwrite the default server url (junglebus.gorillapool.io)
func WithHTTP(serverURL string) ClientOps {
	return func(c *Client) {
		if c != nil {
			transport, _ := transports.NewTransport(
				transports.WithHTTP(serverURL),
				transports.WithDebugging(c.debug),
			)
			c.transport = transport
		}
	}
}

// WithHTTPClient will overwrite the default client with a custom client
func WithHTTPClient(serverURL string, httpClient *http.Client) ClientOps {
	return func(c *Client) {
		if c != nil {
			transport, _ := transports.NewTransport(
				transports.WithHTTPClient(serverURL, httpClient),
				transports.WithDebugging(c.debug),
			)
			c.transport = transport
		}
	}
}

// WithToken will set the token to use in all requests
func WithToken(token string) ClientOps {
	return func(c *Client) {
		if c != nil {
			if c.transport != nil {
				c.transport.SetToken(token)
			}
		}
	}
}

// WithDebugging will set whether to turn debugging on
func WithDebugging(debug bool) ClientOps {
	return func(c *Client) {
		if c != nil {
			c.debug = debug
			if c.transport != nil {
				c.transport.SetDebug(debug)
			}
		}
	}
}

// WithSSL will set whether to use SSL in all communications or not
func WithSSL(useSSL bool) ClientOps {
	return func(c *Client) {
		if c != nil {
			if c.transport != nil {
				c.transport.UseSSL(useSSL)
			}
		}
	}
}

// WithVersion will set the API version to use (v1 is default)
func WithVersion(version string) ClientOps {
	return func(c *Client) {
		if c != nil {
			if c.transport != nil {
				c.transport.SetVersion(version)
			}
		}
	}
}
