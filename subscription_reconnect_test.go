package junglebus

import (
	"context"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GorillaPool/go-junglebus/models"
	"github.com/centrifugal/protocol"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

func TestReconnectAfterWebsocketDisconnect(t *testing.T) {
	var connections int32
	received := make(chan string, 1)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		number := atomic.AddInt32(&connections, 1)
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			decoder := protocol.NewProtobufCommandDecoder(data)
			for {
				command, _ := decoder.Decode()
				if command == nil {
					break
				}
				reply := &protocol.Reply{Id: command.Id}
				switch {
				case command.Connect != nil:
					reply.Connect = &protocol.ConnectResult{Client: "test"}
				case command.Subscribe != nil:
					reply.Subscribe = &protocol.SubscribeResult{}
				case command.Unsubscribe != nil:
					reply.Unsubscribe = &protocol.UnsubscribeResult{}
				default:
					continue
				}
				encoded, err := protocol.NewProtobufReplyEncoder().Encode(reply)
				if err != nil {
					return
				}
				var length [10]byte
				n := binary.PutUvarint(length[:], uint64(len(encoded)))
				if err := conn.WriteMessage(websocket.BinaryMessage, append(length[:n], encoded...)); err != nil {
					return
				}
				// Break an established subscription, rather than failing the handshake.
				if command.Subscribe != nil && strings.Contains(command.Subscribe.Channel, ":100:") {
					if number == 1 {
						return
					}
					data, err := proto.Marshal(&models.TransactionResponse{Id: "resumed", BlockHeight: 100})
					if err != nil {
						return
					}
					push, err := protocol.NewProtobufReplyEncoder().Encode(&protocol.Reply{Push: &protocol.Push{Channel: command.Subscribe.Channel, Pub: &protocol.Publication{Data: data}}})
					if err != nil {
						return
					}
					n := binary.PutUvarint(length[:], uint64(len(push)))
					if err := conn.WriteMessage(websocket.BinaryMessage, append(length[:n], push...)); err != nil {
						return
					}
				}
			}
		}
	}))
	defer server.Close()
	client, err := New(WithHTTP(server.URL), WithToken("test-token"))
	if err != nil {
		t.Fatal(err)
	}
	client.transport.SetToken("test-token")
	connected := make(chan struct{}, 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err = client.SubscribeWithQueue(ctx, "test", 100, 0, EventHandler{
		OnTransaction: func(tx *models.TransactionResponse) { received <- tx.Id },
		OnStatus: func(s *models.ControlResponse) {
			t.Logf("status: %d %s", s.StatusCode, s.Message)
			if s.StatusCode == uint32(StatusConnected) {
				connected <- struct{}{}
			}
		},
		OnError: func(err error) { t.Logf("error: %v", err) },
	}, &SubscribeOptions{QueueSize: 10, LiteMode: true})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		select {
		case <-connected:
		case <-time.After(5 * time.Second):
			t.Fatalf("subscription did not reconnect after websocket disconnect: received %d connected events, %d sockets", i, atomic.LoadInt32(&connections))
		}
	}
	select {
	case id := <-received:
		if id != "resumed" {
			t.Fatalf("unexpected transaction %q", id)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no transaction after reconnect")
	}
	if err := client.Unsubscribe(); err != nil {
		t.Fatal(err)
	}
}
