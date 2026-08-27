package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Nox1KCL/Arbitrage/internal/database/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *botService) HandleUpdate(ctx context.Context, msg Message) error {
	if len(msg.Entities) == 0 || msg.Entities[0].Type != "bot_command" {
		return nil
	}
	entity := msg.Entities[0]

	childCtx, span := s.Observer.Tracer.Start(ctx, "HandleUpdate")
	defer span.End()

	cmd, _, _ := strings.Cut(msg.Text[:entity.Length], "@")
	args := strings.Fields(msg.Text[entity.Length:])
	var text string

	switch cmd {
	case "/subscribe":
		span.AddEvent("Starting subscribe process")

		var err error
		text, err = s.handleSubscribe(childCtx, msg.Chat.ID, args)
		if err != nil {
			span.SetStatus(codes.Error, "trying to register subscribe")
			span.RecordError(err, trace.WithAttributes(attribute.Int("user.id", int(msg.Chat.ID))))

			return fmt.Errorf("handling subscription: %w", err)
		}

	case "/subscriptions":
		span.AddEvent("Starting display of subscriptions process")

		var err error
		text, err = s.handleSubscriptions(childCtx, msg.Chat.ID)
		if err != nil {
			span.SetStatus(codes.Error, "handling display of subscriptions")
			span.RecordError(err, trace.WithAttributes(attribute.Int("user.id", int(msg.Chat.ID))))

			return fmt.Errorf("handling subscriptions: %w", err)
		}

	case "/unsubscribe":
		span.AddEvent("Starting unsubscribe process")

		var err error
		text, err = s.handleUnsubscribe(childCtx, msg.Chat.ID, args)
		if err != nil {
			span.SetStatus(codes.Error, "trying to unsubscribe")
			span.RecordError(err, trace.WithAttributes(attribute.Int("user.id", int(msg.Chat.ID))))

			return fmt.Errorf("handling unsubscribe: %w", err)
		}

	case "/start":
		span.AddEvent("Starting welcome process")

		text = fmt.Sprintf(
			"⚡ Noxie Arbitrage Bot\n\n" +
				"Commands:\n" +
				"/subscribe SYMBOL SPREAD%% VOLUME CHANGE%%\n" +
				"  e.g. /subscribe BTCUSDT 0.5 100 1\n\n" +
				"/subscriptions — show your active subs\n\n" +
				"/unsubscribe SYMBOL\n" +
				"  e.g. /unsubscribe BTCUSDT\n\n" +
				"Start by subscribing to a pair and wait for alerts!",
		)

	default:
		span.AddEvent("Unknown command")
		text = "Sorry! I don't know that command."
	}

	err := s.sendMessage(childCtx, msg.Chat.ID, text)
	if err != nil {
		span.SetStatus(codes.Error, "trying to send message")
		span.RecordError(err, trace.WithAttributes(attribute.Int("user.id", int(msg.Chat.ID))))

		return fmt.Errorf("sending message: %w", err)
	}
	s.metrics.sentMessages.Add(childCtx, 1)

	return nil
}

func (s *botService) handleSubscribe(ctx context.Context, id int64, args []string) (string, error) {
	if len(args) != 4 {
		return "You must specify 4 arguments!", nil
	}
	childCtx, span := s.Observer.Tracer.Start(ctx, "HandlingSubscribe")
	defer span.End()

	symbol := strings.ToUpper(args[0])
	minSpread, err1 := strconv.ParseFloat(args[1], 64)
	minVolume, err2 := strconv.ParseFloat(args[2], 64)
	minPriceChange, err3 := strconv.ParseFloat(args[3], 64)
	if err := errors.Join(err1, err2, err3); err != nil {
		span.SetStatus(codes.Error, "could not parse values to float")
		span.RecordError(err, trace.WithAttributes(
			attribute.Int("user.id", int(id))),
		)

		return "", fmt.Errorf("could not convert number to float: %w", err)
	}

	if minSpread <= 0 || minVolume <= 0 || minPriceChange <= 0 {
		err := fmt.Errorf("one or many arguments are equal or below zero")
		span.SetStatus(codes.Error, "one or many of values is equal or below zero")
		span.RecordError(err, trace.WithAttributes(
			attribute.Int("user.id", int(id))),
		)

		return "Values must be greater that zero.", nil
	}

	result := s.db.Create(&models.Subscription{
		TelegramChatID:        id,
		Symbol:                symbol,
		MinSpreadPercent:      minSpread,
		MinVolume:             minVolume,
		MinPriceChangePercent: minPriceChange,
	})

	if result.Error != nil {
		span.SetStatus(codes.Error, "could not register subscription")
		span.RecordError(result.Error, trace.WithAttributes(
			attribute.Int("user.id", int(id)),
			attribute.Int("rowsAffected", int(result.RowsAffected)),
		))

		s.metrics.dbErrors.Add(childCtx, 1)
		return "You already have the same subscription!", nil
	}

	text := fmt.Sprintf("You are now subscribed to %s", symbol)
	if _, err := s.client.ReloadSubscriptions(childCtx, &emptypb.Empty{}); err != nil {
		span.SetStatus(codes.Error, "reloading subscriptions")
		span.RecordError(err, trace.WithAttributes(
			attribute.Int("user.id", int(id))),
		)

		return "", fmt.Errorf("reloading subscriptions: %w", err)
	}
	s.metrics.activeSubscriptions.Add(childCtx, 1)

	return text, nil
}

func (s *botService) handleSubscriptions(ctx context.Context, id int64) (string, error) {
	childCtx, span := s.Observer.Tracer.Start(ctx, "HandlingSubscriptions")
	defer span.End()

	var subs []models.Subscription
	result := s.db.Where("telegram_chat_id = ?", id).Find(&subs)
	if result.Error != nil {
		span.SetStatus(codes.Error, "could not find subscription")
		span.RecordError(result.Error, trace.WithAttributes(
			attribute.Int("user.id", int(id)),
			attribute.Int("rowsAffected", int(result.RowsAffected)),
		))

		s.metrics.dbErrors.Add(childCtx, 1)
		return "", fmt.Errorf("finding fields with ID-%d: %w", id, result.Error)
	}
	if len(subs) == 0 {
		span.SetStatus(codes.Error, "user subs is equal zero")
		return "You don't have active subs :(", nil
	}

	var lines []string
	for _, s := range subs {
		line := fmt.Sprintf("%s | spread ≥ %.2f%% | vol ≥ %.0f | Δ ≥ %.1f%%",
			s.Symbol, s.MinSpreadPercent, s.MinVolume, s.MinPriceChangePercent)
		lines = append(lines, line)
	}
	text := fmt.Sprintf("Your subscriptions:\n\n%s\n\n%d active", strings.Join(lines, "\n"), len(subs))

	return text, nil
}

func (s *botService) handleUnsubscribe(ctx context.Context, id int64, args []string) (string, error) {
	if len(args) != 1 {
		return "You forgot to say Ticker(symbol)!!", nil
	}

	childCtx, span := s.Observer.Tracer.Start(ctx, "HandlingUnsubscribe")
	defer span.End()

	symbol := strings.ToUpper(args[0])
	result := s.db.Where("telegram_chat_id = ? AND symbol = ?", id, symbol).Delete(&models.Subscription{})
	if result.Error != nil {
		span.SetStatus(codes.Error, "could not delete subscription")
		span.RecordError(result.Error, trace.WithAttributes(
			attribute.Int("user.id", int(id))),
		)

		s.metrics.dbErrors.Add(childCtx, 1)
		return "", fmt.Errorf("deleting sub with ID-%d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		err := fmt.Errorf("db returns 0 rows")
		span.SetStatus(codes.Error, "db rowsAffected is equal zero")
		span.RecordError(err, trace.WithAttributes(
			attribute.Int("user.id", int(id)),
		))
		return "I could not find this sub..", nil
	}

	text := fmt.Sprintf("You are now no more subscribed to %s", symbol)
	if _, err := s.client.ReloadSubscriptions(childCtx, &emptypb.Empty{}); err != nil {
		span.SetStatus(codes.Error, "reloading subscriptions")
		span.RecordError(err, trace.WithAttributes(
			attribute.Int("user.id", int(id))),
		)

		return "", fmt.Errorf("reloading subscriptions: %w", err)
	}
	s.metrics.activeSubscriptions.Add(childCtx, -1)

	return text, nil
}
