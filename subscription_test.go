package junglebus

import (
	"context"
	"testing"

	"github.com/GorillaPool/go-junglebus/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func newTestEventHandler() (EventHandler, *bool, *bool, *bool, *bool) {
	onTransactionCalled := false
	onMempoolCalled := false
	onStatusCalled := false
	onErrorCalled := false

	return EventHandler{
		OnTransaction: func(tx *models.TransactionResponse) {
			onTransactionCalled = true
		},
		OnMempool: func(tx *models.TransactionResponse) {
			onMempoolCalled = true
		},
		OnStatus: func(status *models.ControlResponse) {
			onStatusCalled = true
		},
		OnError: func(err error) {
			onErrorCalled = true
		},
		ctx:   context.Background(),
		debug: false,
	}, &onTransactionCalled, &onMempoolCalled, &onStatusCalled, &onErrorCalled
}

func TestSubscription_HandlePubChan(t *testing.T) {
	t.Run("handle control message", func(t *testing.T) {
		handler, _, _, onStatusCalled, _ := newTestEventHandler()
		sub := &Subscription{
			EventHandler: handler,
			pubChan:      make(chan *pubEvent),
		}

		// Create a control message
		status := &models.ControlResponse{
			StatusCode: uint32(SubscriptionBlockDone),
			Block:      100,
		}
		statusData, err := proto.Marshal(status)
		require.NoError(t, err)

		// Start handling messages
		go sub.handlePubChan(&SubscribeOptions{})

		// Send a control message
		sub.addToQueue(&pubEvent{
			Channel: "control",
			Data:    statusData,
		})

		// Close the channel to stop the handler
		close(sub.pubChan)
		sub.wg.Wait()

		assert.True(t, *onStatusCalled)
	})

	t.Run("handle transaction message", func(t *testing.T) {
		handler, onTransactionCalled, _, _, _ := newTestEventHandler()
		client, err := New()
		require.NoError(t, err)

		sub := &Subscription{
			EventHandler: handler,
			client:       client,
			pubChan:      make(chan *pubEvent),
		}

		// Create a transaction message
		tx := &models.TransactionResponse{
			Id:          "test-tx",
			BlockHeight: 100,
			Transaction: []byte("test-tx-data"),
		}
		txData, err := proto.Marshal(tx)
		require.NoError(t, err)

		// Start handling messages
		go sub.handlePubChan(&SubscribeOptions{})

		// Send a transaction message
		sub.addToQueue(&pubEvent{
			Channel: "main",
			Data:    txData,
		})

		// Close the channel to stop the handler
		close(sub.pubChan)
		sub.wg.Wait()

		assert.True(t, *onTransactionCalled)
	})

	t.Run("handle mempool message", func(t *testing.T) {
		handler, _, onMempoolCalled, _, _ := newTestEventHandler()
		client, err := New()
		require.NoError(t, err)

		sub := &Subscription{
			EventHandler: handler,
			client:       client,
			pubChan:      make(chan *pubEvent),
		}

		// Create a mempool message
		tx := &models.TransactionResponse{
			Id:          "test-tx",
			BlockHeight: 0,
			Transaction: []byte("test-tx-data"),
		}
		txData, err := proto.Marshal(tx)
		require.NoError(t, err)

		// Start handling messages
		go sub.handlePubChan(&SubscribeOptions{})

		// Send a mempool message
		sub.addToQueue(&pubEvent{
			Channel: "mempool",
			Data:    txData,
		})

		// Close the channel to stop the handler
		close(sub.pubChan)
		sub.wg.Wait()

		assert.True(t, *onMempoolCalled)
	})
}
