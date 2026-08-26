package processing_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Nox1KCL/Arbitrage/internal/processing"
	"github.com/Nox1KCL/Arbitrage/internal/transport"
	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
	"google.golang.org/grpc"
)

type mockDataServiceClient struct {
	sendUserFunc func(ctx context.Context, in *pb.AlertNotification, opts ...grpc.CallOption) (*pb.Ack, error)
}

func (m *mockDataServiceClient) SendUser(ctx context.Context, in *pb.AlertNotification, opts ...grpc.CallOption) (*pb.Ack, error) {
	if m.sendUserFunc != nil {
		return m.sendUserFunc(ctx, in, opts...)
	}
	return &pb.Ack{Status: true}, nil
}

func TestSendingServer(t *testing.T) {
	obs := newDummyTelemetry()

	t.Run("successful message delivery", func(t *testing.T) {
		receivedChan := make(chan *pb.AlertNotification, 1)
		client := &mockDataServiceClient{
			sendUserFunc: func(ctx context.Context, in *pb.AlertNotification, opts ...grpc.CallOption) (*pb.Ack, error) {
				receivedChan <- in
				return &pb.Ack{Status: true}, nil
			},
		}

		server := &processing.SendingServer{
			Client: client,
			Obs:    obs,
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		payload := make(chan []*transport.FormedMessage, 5)
		go server.Sending(ctx, payload)

		payload <- []*transport.FormedMessage{
			{TelegramUserID: 12345, Text: "Test Alert"},
		}

		select {
		case receivedReq := <-receivedChan:
			if receivedReq.TelegramChatId != 12345 || receivedReq.Text != "Test Alert" {
				t.Errorf("unexpected request: %+v", receivedReq)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for grpc send")
		}
	})

	t.Run("grpc error does not crash server", func(t *testing.T) {
		calledChan := make(chan struct{}, 1)
		client := &mockDataServiceClient{
			sendUserFunc: func(ctx context.Context, in *pb.AlertNotification, opts ...grpc.CallOption) (*pb.Ack, error) {
				calledChan <- struct{}{}
				return nil, errors.New("grpc unavailable")
			},
		}

		server := &processing.SendingServer{
			Client: client,
			Obs:    obs,
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		payload := make(chan []*transport.FormedMessage, 5)
		go server.Sending(ctx, payload)

		payload <- []*transport.FormedMessage{
			{TelegramUserID: 12345, Text: "Test Alert"},
		}

		select {
		case <-calledChan:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for grpc send")
		}
	})

	t.Run("context cancellation stops sending loop", func(t *testing.T) {
		client := &mockDataServiceClient{}
		server := &processing.SendingServer{
			Client: client,
			Obs:    obs,
		}

		ctx, cancel := context.WithCancel(context.Background())
		payload := make(chan []*transport.FormedMessage, 5)

		done := make(chan struct{})
		go func() {
			server.Sending(ctx, payload)
			close(done)
		}()

		cancel()

		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("server did not stop on context cancellation")
		}
	})
}
