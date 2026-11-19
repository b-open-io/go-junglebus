package junglebus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/b-open-io/go-junglebus/models"
	"github.com/centrifugal/centrifuge"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// testServer is an in-process centrifuge server that behaves like the
// JungleBus server for the parts that matter to reconnect handling.
type testServer struct {
	t      *testing.T
	node   *centrifuge.Node
	http   *httptest.Server
	host   string
	delay  time.Duration // how long the main channel OnSubscribe takes (JungleBus does a DB lookup)
	reject func(channel string) error

	mu         sync.Mutex
	subscribes map[string]int // subscribe commands that reached OnSubscribe, per channel
	duplicates map[string]int // subscribe commands rejected as already subscribed on the same connection
	clients    []*centrifuge.Client
}

func newTestServer(t *testing.T, delay time.Duration) *testServer {
	t.Helper()
	ts := &testServer{t: t, delay: delay, subscribes: map[string]int{}, duplicates: map[string]int{}}

	// A duplicate subscribe on the same connection is rejected inside
	// centrifuge before OnSubscribe runs; its log entry is the only trace.
	node, err := centrifuge.New(centrifuge.Config{
		LogLevel: centrifuge.LogLevelInfo,
		LogHandler: func(e centrifuge.LogEntry) {
			if e.Message != "client already subscribed on channel" {
				return
			}
			channel, _ := e.Fields["channel"].(string)
			ts.mu.Lock()
			ts.duplicates[channel]++
			ts.mu.Unlock()
		},
	})
	require.NoError(t, err)
	ts.node = node

	node.OnConnecting(func(ctx context.Context, e centrifuge.ConnectEvent) (centrifuge.ConnectReply, error) {
		return centrifuge.ConnectReply{Credentials: &centrifuge.Credentials{UserID: ""}}, nil
	})
	node.OnConnect(func(client *centrifuge.Client) {
		ts.mu.Lock()
		ts.clients = append(ts.clients, client)
		ts.mu.Unlock()

		client.OnSubscribe(func(e centrifuge.SubscribeEvent, cb centrifuge.SubscribeCallback) {
			ts.mu.Lock()
			ts.subscribes[e.Channel]++
			reject := ts.reject
			ts.mu.Unlock()

			if reject != nil {
				if err := reject(e.Channel); err != nil {
					cb(centrifuge.SubscribeReply{}, err)
					return
				}
			}
			if isMainChannel(e.Channel) && ts.delay > 0 {
				// Reply asynchronously, like a handler that does I/O first.
				// centrifuge keeps reading commands meanwhile, so a duplicate
				// subscribe is rejected before this one succeeds.
				go func() {
					time.Sleep(ts.delay)
					cb(centrifuge.SubscribeReply{}, nil)
				}()
				return
			}
			cb(centrifuge.SubscribeReply{}, nil)
		})
	})
	require.NoError(t, node.Run())

	mux := http.NewServeMux()
	mux.Handle("/connection/websocket", centrifuge.NewWebsocketHandler(node, centrifuge.WebsocketConfig{
		CheckOrigin: func(r *http.Request) bool { return true },
	}))
	ts.http = httptest.NewServer(mux)
	ts.host = strings.TrimPrefix(ts.http.URL, "http://")

	t.Cleanup(func() {
		ts.http.Close()
		_ = node.Shutdown(context.Background())
	})
	return ts
}

func isMainChannel(channel string) bool {
	parts := strings.Split(channel, ":")
	return len(parts) == 4 // {lite|query}:{id}:{block}:{page}
}

func (ts *testServer) subscribeCount(channel string) int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.subscribes[channel]
}

func (ts *testServer) duplicateCount(channel string) int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.duplicates[channel]
}

// forceReconnect disconnects every connected client with a reconnect advice,
// which is what clients see when nginx in front of JungleBus drops them.
func (ts *testServer) forceReconnect() {
	ts.mu.Lock()
	clients := ts.clients
	ts.clients = nil
	ts.mu.Unlock()
	for _, c := range clients {
		c.Disconnect(centrifuge.DisconnectForceReconnect)
	}
}

func (ts *testServer) publish(channel string, msg proto.Message) {
	data, err := proto.Marshal(msg)
	require.NoError(ts.t, err)
	_, err = ts.node.Publish(channel, data)
	require.NoError(ts.t, err)
}

// publishUntilReceived publishes a transaction on channel until the client
// delivers it to OnTransaction. Publications sent before the subscription is
// fully established are lost, so a single publish would make the test racy.
func (ts *testServer) publishUntilReceived(channel, txID string, received <-chan *models.TransactionResponse, timeout time.Duration) bool {
	deadline := time.After(timeout)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		ts.publish(channel, &models.TransactionResponse{Id: txID})
		select {
		case tx := <-received:
			if tx.Id == txID {
				return true
			}
		case <-tick.C:
		case <-deadline:
			return false
		}
	}
}

func newTestSubscription(t *testing.T, ts *testServer, subID string, fromBlock uint64) (*Subscription, chan *models.TransactionResponse, chan error) {
	t.Helper()
	return newTestSubscriptionWithStatus(t, ts, subID, fromBlock, func(*models.ControlResponse) {})
}

func newTestSubscriptionWithStatus(t *testing.T, ts *testServer, subID string, fromBlock uint64, onStatus func(*models.ControlResponse)) (*Subscription, chan *models.TransactionResponse, chan error) {
	t.Helper()
	client, err := New(WithHTTP(ts.host), WithToken("test-token"), WithSSL(false))
	require.NoError(t, err)

	received := make(chan *models.TransactionResponse, 100)
	errs := make(chan error, 100)
	sub, err := client.SubscribeWithQueue(context.Background(), subID, fromBlock, 0, EventHandler{
		OnTransaction: func(tx *models.TransactionResponse) { received <- tx },
		OnStatus:      onStatus,
		OnError:       func(err error) { errs <- err },
	}, &SubscribeOptions{QueueSize: 100, LiteMode: true})
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Unsubscribe() })
	return sub, received, errs
}

func waitFor(t *testing.T, timeout time.Duration, msg string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", msg)
}

// TestReconnect_MainChannelSubscribedOnceAndDelivers reproduces the production
// failure: after a reconnect the main channel was subscribed twice on the same
// connection, the server's "already subscribed" reply for the duplicate arrived
// first, and the client silently dropped every publication afterwards.
func TestReconnect_MainChannelSubscribedOnceAndDelivers(t *testing.T) {
	ts := newTestServer(t, 200*time.Millisecond)
	const subID = "reconnect-test"

	sub, received, _ := newTestSubscription(t, ts, subID, 100)

	firstMain := fmt.Sprintf("lite:%s:100:0", subID)
	require.True(t, ts.publishUntilReceived(firstMain, "tx-before", received, 5*time.Second),
		"transaction on initial main channel not delivered")

	// Server finishes block 100; the client should move to block 101.
	ts.publish(fmt.Sprintf("lite:%s:control", subID), &models.ControlResponse{
		StatusCode: uint32(SubscriptionBlockDone),
		Status:     "block-done",
		Block:      100,
	})
	waitFor(t, 5*time.Second, "position to advance to 101", func() bool {
		block, _ := sub.Position()
		return block == 101
	})

	ts.forceReconnect()

	nextMain := fmt.Sprintf("lite:%s:101:0", subID)
	waitFor(t, 10*time.Second, "subscribe on "+nextMain, func() bool { return ts.subscribeCount(nextMain) > 0 })
	// Give a would-be duplicate time to show up.
	time.Sleep(2 * ts.delay)
	require.Equal(t, 1, ts.subscribeCount(nextMain), "main channel must be subscribed exactly once after reconnect")
	require.Zero(t, ts.duplicateCount(nextMain), "main channel must not be subscribed twice on the same connection")

	require.True(t, ts.publishUntilReceived(nextMain, "tx-after", received, 5*time.Second),
		"transaction on main channel after reconnect not delivered")
}

// TestReconnect_RecreatesUnsubscribedMainChannel covers a main channel that
// ended up unsubscribed client-side (non-temporary subscribe error). The error
// must be reported, and the next reconnect must subscribe the channel again
// even though the position, and therefore the channel name, has not changed.
func TestReconnect_RecreatesUnsubscribedMainChannel(t *testing.T) {
	ts := newTestServer(t, 0)
	const subID = "recreate-test"
	mainChannel := fmt.Sprintf("lite:%s:100:0", subID)

	var rejected sync.Once
	ts.reject = func(channel string) error {
		var err error
		if channel == mainChannel {
			rejected.Do(func() { err = centrifuge.ErrorPermissionDenied })
		}
		return err
	}

	_, received, errs := newTestSubscription(t, ts, subID, 100)

	select {
	case err := <-errs:
		require.Contains(t, err.Error(), mainChannel)
	case <-time.After(5 * time.Second):
		t.Fatal("expected an error for the unsubscribed main channel")
	}
	require.Equal(t, 1, ts.subscribeCount(mainChannel))

	ts.forceReconnect()

	waitFor(t, 10*time.Second, "main channel to be resubscribed", func() bool { return ts.subscribeCount(mainChannel) >= 2 })
	require.True(t, ts.publishUntilReceived(mainChannel, "tx-recovered", received, 5*time.Second),
		"transaction not delivered after main channel was recreated")

	// No further errors expected once recovered.
	select {
	case err := <-errs:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("unexpected error after recovery: %v", err)
		}
	default:
	}
}

// TestReconnect_UsesPositionAfterQueuedEvents covers events from the old
// connection that are still queued when the connection drops. The reconnect
// must resume after them, not from the position at the moment of the drop,
// which would replay the blocks they completed.
func TestReconnect_UsesPositionAfterQueuedEvents(t *testing.T) {
	ts := newTestServer(t, 0)
	const subID = "queued-test"

	held := make(chan struct{})
	release := make(chan struct{})
	var holdOnce sync.Once
	sub, received, _ := newTestSubscriptionWithStatus(t, ts, subID, 100, func(status *models.ControlResponse) {
		// Hold the event processor on the first block-done, so the next one
		// stays queued.
		if status.StatusCode == uint32(SubscriptionBlockDone) {
			holdOnce.Do(func() {
				close(held)
				<-release
			})
		}
	})

	firstMain := fmt.Sprintf("lite:%s:100:0", subID)
	require.True(t, ts.publishUntilReceived(firstMain, "tx-before", received, 5*time.Second))

	control := fmt.Sprintf("lite:%s:control", subID)
	ts.publish(control, &models.ControlResponse{StatusCode: uint32(SubscriptionBlockDone), Status: "block-done", Block: 100})
	<-held
	ts.publish(control, &models.ControlResponse{StatusCode: uint32(SubscriptionBlockDone), Status: "block-done", Block: 101})
	waitFor(t, 5*time.Second, "second block-done to be queued", func() bool { return sub.eventQueue.Len() == 1 })

	ts.forceReconnect()
	time.Sleep(200 * time.Millisecond) // reconnect starts while the event is still queued
	close(release)

	resumed := fmt.Sprintf("lite:%s:102:0", subID)
	waitFor(t, 10*time.Second, "subscribe on "+resumed, func() bool { return ts.subscribeCount(resumed) > 0 })
	require.Zero(t, ts.subscribeCount(fmt.Sprintf("lite:%s:101:0", subID)), "must not resume from a position before queued block-done events")
	require.True(t, ts.publishUntilReceived(resumed, "tx-after", received, 5*time.Second))
}

// TestReconnect_WhileMainSubscribePending disconnects while the main channel's
// subscribe is still waiting for its reply. The pending request fails with the
// old connection and may schedule a retry; that retry must not overlap the new
// connection's resubscribe and subscribe the channel twice.
func TestReconnect_WhileMainSubscribePending(t *testing.T) {
	ts := newTestServer(t, 500*time.Millisecond)
	const subID = "pending-test"
	mainChannel := fmt.Sprintf("lite:%s:100:0", subID)

	_, received, _ := newTestSubscription(t, ts, subID, 100)
	waitFor(t, 5*time.Second, "first main subscribe", func() bool { return ts.subscribeCount(mainChannel) == 1 })
	ts.forceReconnect() // the first subscribe's reply is still pending

	waitFor(t, 10*time.Second, "main channel resubscribed", func() bool { return ts.subscribeCount(mainChannel) >= 2 })
	time.Sleep(3 * ts.delay) // let any overlapping retry show up
	require.Zero(t, ts.duplicateCount(mainChannel), "main channel must not be subscribed twice on the same connection")
	require.True(t, ts.publishUntilReceived(mainChannel, "tx-after", received, 5*time.Second),
		"transaction not delivered after reconnecting during a pending subscribe")
}
