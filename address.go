package junglebus

import (
	"context"
	"errors"

	"github.com/b-open-io/go-junglebus/models"
)

// GetAddressTransactions get transaction meta data for the given address
func (jb *Client) GetAddressTransactions(ctx context.Context, address string, fromHeight uint32) ([]*models.AddressTx, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}
	if address == "" {
		return nil, errors.New("address cannot be empty")
	}
	return jb.transport.GetAddressTransactions(ctx, address, fromHeight)
}

// GetAddressTransactionDetails get full transaction data for the given address
func (jb *Client) GetAddressTransactionDetails(ctx context.Context, address string, fromHeight uint32) ([]*models.Transaction, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}
	if address == "" {
		return nil, errors.New("address cannot be empty")
	}
	return jb.transport.GetAddressTransactionDetails(ctx, address, fromHeight)
}
