package bot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	telemetry "github.com/Nox1KCL/Arbitrage/internal/observer"
	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func SetTelegramAPI(url string) func() {
	old := telegramAPI
	telegramAPI = url
	return func() { telegramAPI = old }
}

func (s *botService) SendMessage(ctx context.Context, id int64, text string) error {
	return s.sendMessage(ctx, id, text)
}

type NewTestMockProcessingClient struct {
	reloadErr error
}

func NewTestEntities(entityType string, offset int, length int) []Entity {
	return []Entity{
		{
			Type:   entityType,
			Offset: offset,
			Length: length,
		},
	}
}

func NewTestChat(id int64) Chat {
	return Chat{ID: id}
}

func NewTestMessage(text string, chat Chat, entities []Entity) Message {
	return Message{
		Text:     text,
		Chat:     chat,
		Entities: entities,
	}
}

func (m *NewTestMockProcessingClient) ReloadSubscriptions(
	ctx context.Context,
	in *emptypb.Empty,
	opts ...grpc.CallOption,
) (*pb.ReloadSubscriptionsResponse, error) {
	return &pb.ReloadSubscriptionsResponse{}, m.reloadErr
}

func NewTestDummyTelemetry() *telemetry.Observe {
	tp := tracenoop.NewTracerProvider()
	mp := metricnoop.NewMeterProvider()
	return &telemetry.Observe{
		Tracer: tp.Tracer("test"),
		Meter:  mp.Meter("test"),
	}
}

func NewTestBotService(t *testing.T, token string, onSendMessage func(chatID int64, text string)) *botService {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ChatID int64  `json:"chat_id"`
			Text   string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if onSendMessage != nil {
			onSendMessage(req.ChatID, req.Text)
		}
		w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	oldAPI := telegramAPI
	telegramAPI = server.URL
	t.Cleanup(func() {
		server.Close()
		telegramAPI = oldAPI
	})

	obs := NewTestDummyTelemetry()
	client := &NewTestMockProcessingClient{}

	service, err := NewBotService(obs, nil, token, client)
	if err != nil {
		t.Fatalf("failed to init bot service: %v", err)
	}

	return service
}
