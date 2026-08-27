package bot

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/codes"
)

func (s *botService) Polling(ctx context.Context) error {
	var offset int64 = 0
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancel signal")
		default:
		}

		s.PollOnce(context.Background(), &offset)
	}
}

func (s *botService) PollOnce(ctx context.Context, offset *int64) {
	ctx, span := s.Observer.Tracer.Start(ctx, "BotPolling")
	defer span.End()

	updates, err := s.GetUpdates(ctx, offset)
	if err != nil {
		span.SetStatus(codes.Error, "Trouble to get updates")
		span.RecordError(err)
		blog.ErrorContext(ctx, "Trying to get updates", "error", err)

		time.Sleep(2 * time.Second)
	}

	for _, update := range updates {
		if err := s.HandleUpdate(ctx, update.Message); err != nil {
			span.SetStatus(codes.Error, "Trouble to handle update")
			span.RecordError(err)
			blog.ErrorContext(ctx, "Handling update", "error", err)
		}
		*offset = int64(update.UpdateID + 1)
	}
}
