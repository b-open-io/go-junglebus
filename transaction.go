package junglebus

import (
	"context"
	"errors"

	"github.com/GorillaPool/go-junglebus/models"
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
