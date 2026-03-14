package junglebus

import (
	"context"
	"errors"

	"github.com/b-open-io/go-junglebus/models"
)

func (jb *Client) GetTransaction(ctx context.Context, txID string) (*models.Transaction, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}
	if txID == "" {
		return nil, errors.New("transaction ID cannot be empty")
	}
	return jb.transport.GetTransaction(ctx, txID)
}

func (jb *Client) GetRawTransaction(ctx context.Context, txID string) ([]byte, error) {
	return jb.transport.GetRawTransaction(ctx, txID)
}

func (jb *Client) GetBeef(ctx context.Context, txID string) ([]byte, error) {
	return jb.transport.GetBeef(ctx, txID)
}

func (jb *Client) GetProof(ctx context.Context, txID string) ([]byte, error) {
	return jb.transport.GetProof(ctx, txID)
}
