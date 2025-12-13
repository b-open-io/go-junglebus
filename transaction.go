package junglebus

import (
	"context"

	"github.com/b-open-io/go-junglebus/models"
)

func (jb *Client) GetTransaction(ctx context.Context, txID string) (*models.Transaction, error) {
	return jb.transport.GetTransaction(ctx, txID)
}
