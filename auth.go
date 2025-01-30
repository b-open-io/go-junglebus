package junglebus

import (
	"context"
	"errors"
)

// Login will authenticate with the server using username and password
func (jb *Client) Login(ctx context.Context, username string, password string) error {
	if ctx == nil {
		return errors.New("context cannot be nil")
	}
	if username == "" {
		return errors.New("username cannot be empty")
	}
	if password == "" {
		return errors.New("password cannot be empty")
	}
	return jb.transport.Login(ctx, username, password)
}

// GetSubscriptionToken will get a token based on the subscription ID
func (jb *Client) GetSubscriptionToken(ctx context.Context, subscriptionID string) (string, error) {
	if ctx == nil {
		return "", errors.New("context cannot be nil")
	}
	if subscriptionID == "" {
		return "", errors.New("subscription ID cannot be empty")
	}
	return jb.transport.GetSubscriptionToken(ctx, subscriptionID)
}

// RefreshToken will refresh the current token
func (jb *Client) RefreshToken(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", errors.New("context cannot be nil")
	}
	return jb.transport.RefreshToken(ctx)
}
