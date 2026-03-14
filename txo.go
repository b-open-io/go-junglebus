package junglebus

import (
	"context"
	"errors"
)

// GetTxo retrieves the raw transaction output data for the given outpoint
func (jb *Client) GetTxo(ctx context.Context, txID string, vout uint32) ([]byte, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}
	if txID == "" {
		return nil, errors.New("transaction ID cannot be empty")
	}
	return jb.transport.GetTxo(ctx, txID, vout)
}

// GetSpend retrieves the spending transaction ID for the given outpoint
func (jb *Client) GetSpend(ctx context.Context, txID string, vout uint32) ([]byte, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}
	if txID == "" {
		return nil, errors.New("transaction ID cannot be empty")
	}
	return jb.transport.GetSpend(ctx, txID, vout)
}
