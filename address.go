package junglebus

import (
	"context"

	"github.com/b-open-io/go-junglebus/models"
)

// GetAddressTransactions get transaction meta data for the given address
// from is optional - pass 0 to get all transactions, or a block height to start from
func (jb *Client) GetAddressTransactions(ctx context.Context, address string, from uint32) ([]*models.Address, error) {
	return jb.transport.GetAddressTransactions(ctx, address, from)
}

// GetAddressTransactionDetails get full transaction data for the given address
func (jb *Client) GetAddressTransactionDetails(ctx context.Context, address string) ([]*models.Transaction, error) {
	return jb.transport.GetAddressTransactionDetails(ctx, address)
}
