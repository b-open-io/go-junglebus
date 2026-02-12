package junglebus

import (
	"context"
	"errors"

	"github.com/b-open-io/go-junglebus/models"
)

// GetBlockHeader get the block header for the given block hash or height
func (jb *Client) GetBlockHeader(ctx context.Context, block string) (*models.BlockHeader, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}
	if block == "" {
		return nil, errors.New("block cannot be empty")
	}
	return jb.transport.GetBlockHeader(ctx, block)
}

// GetBlockHeaders get block headers starting from the given block hash or height
func (jb *Client) GetBlockHeaders(ctx context.Context, fromBlock string, limit uint) ([]*models.BlockHeader, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}
	if fromBlock == "" {
		return nil, errors.New("fromBlock cannot be empty")
	}
	return jb.transport.GetBlockHeaders(ctx, fromBlock, limit)
}

// GetChainTip get the current chain tip block header
func (jb *Client) GetChainTip(ctx context.Context) (*models.BlockHeader, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}
	return jb.transport.GetChainTip(ctx)
}
