package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Nox1KCL/Arbitrage/internal/database/models"
	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
)

func HandleUpdate(ctx context.Context, db *gorm.DB, client pb.ProcessingServiceClient, msg Message, token string) error {
	if len(msg.Entities) == 0 || msg.Entities[0].Type != "bot_command" {
		return nil
	}
	entity := msg.Entities[0]

	cmd, _, _ := strings.Cut(msg.Text[:entity.Length], "@")
	args := strings.Fields(msg.Text[entity.Length:])

	switch cmd {
	case "/subscribe":
		text, err := handleSubscribe(msg.Chat.ID, args, db)
		if err != nil {
			return fmt.Errorf("handling subscription: %w", err)
		}
		sendMessage(ctx, token, msg.Chat.ID, text)
		if _, err := client.ReloadSubscriptions(ctx, &emptypb.Empty{}); err != nil {
			return fmt.Errorf("reloading subscriptions: %w", err)
		}

	case "/subscriptions":
		text, err := handleSubscriptions(msg.Chat.ID, db)
		if err != nil {
			return fmt.Errorf("handling subscriptions: %w", err)
		}
		sendMessage(ctx, token, msg.Chat.ID, text)

	case "/unsubscribe":
		text, err := handleUnsubscribe(msg.Chat.ID, args, db)
		if err != nil {
			return fmt.Errorf("handling unsubscribe: %w", err)
		}
		sendMessage(ctx, token, msg.Chat.ID, text)
		if _, err := client.ReloadSubscriptions(ctx, &emptypb.Empty{}); err != nil {
			return fmt.Errorf("reloading subscriptions: %w", err)
		}
	case "/start":
		text := fmt.Sprintf(
			"⚡ Noxie Arbitrage Bot\n\n" +
				"Commands:\n" +
				"/subscribe SYMBOL SPREAD%% VOLUME CHANGE%%\n" +
				"  e.g. /subscribe BTCUSDT 0.5 100 1\n\n" +
				"/subscriptions — show your active subs\n\n" +
				"/unsubscribe SYMBOL\n" +
				"  e.g. /unsubscribe BTCUSDT\n\n" +
				"Start by subscribing to a pair and wait for alerts!",
		)
		sendMessage(ctx, token, msg.Chat.ID, text)

	default:
		sendMessage(ctx, token, msg.Chat.ID, "I don't know that commad.")
	}
	return nil
}

func handleSubscribe(chatID int64, args []string, db *gorm.DB) (string, error) {
	if len(args) != 4 {
		return "You must specify 4 arguments!", nil
	}

	symbol := strings.ToUpper(args[0])
	minSpread, err1 := strconv.ParseFloat(args[1], 64)
	minVolume, err2 := strconv.ParseFloat(args[2], 64)
	minPriceChange, err3 := strconv.ParseFloat(args[3], 64)
	if err := errors.Join(err1, err2, err3); err != nil {
		return "", fmt.Errorf("could not convert number to float: %w", err)
	}

	if minSpread <= 0 || minVolume <= 0 || minPriceChange <= 0 {
		return "Some of your number(s) is 0 or equal.", nil
	}
	result := db.Create(&models.Subscription{
		TelegramChatID:        chatID,
		Symbol:                symbol,
		MinSpreadPercent:      minSpread,
		MinVolume:             minVolume,
		MinPriceChangePercent: minPriceChange,
	})
	if result.Error != nil {
		return "You already have the same subscription!", nil
	}

	text := fmt.Sprintf("You are tune on %s", symbol)
	return text, nil
}

func handleSubscriptions(chatID int64, db *gorm.DB) (string, error) {
	var subs []models.Subscription
	result := db.Where("telegram_chat_id = ?", chatID).Find(&subs)
	if result.Error != nil {
		return "", fmt.Errorf("finding fields with ID-%d: %w", chatID, result.Error)
	}
	if len(subs) == 0 {
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

func handleUnsubscribe(chatID int64, args []string, db *gorm.DB) (string, error) {
	if len(args) != 1 {
		return "You forgot to say Ticker(symbol)!!", nil
	}
	symbol := strings.ToUpper(args[0])
	result := db.Where("telegram_chat_id = ? AND symbol = ?", chatID, symbol).Delete(&models.Subscription{})
	if result.Error != nil {
		return "", fmt.Errorf("deleting sub with ID-%d: %w", chatID, result.Error)
	}
	if result.RowsAffected == 0 {
		return "I could not find this sub..", nil
	}

	text := fmt.Sprintf("You are no more tune on %s", symbol)
	return text, nil
}
