package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Nox1KCL/Arbitrage/internal/database"
	"github.com/Nox1KCL/Arbitrage/internal/database/models"
	"github.com/Nox1KCL/Arbitrage/internal/syncutils"
	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
)

var blog = slog.With("service", "bot")

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	token := os.Getenv("TELEGRAM_BOT_API")
	if token == "" {
		slog.Error("TELEGRAM_BOT_API is not set")
		os.Exit(1)
	}

	conn, err := grpc.NewClient("processing-go:50050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		blog.Error("Could not connect to client", "error", err)
		return
	}
	defer conn.Close()

	client := pb.NewProcessingServiceClient(conn)

	db, err := database.Connect()
	if err != nil {
		slog.Error("Could not connect to db", "error", err)
		os.Exit(1)
	}
	if err := db.AutoMigrate(&models.Subscription{}); err != nil {
		blog.Error("Could not create a table", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	var wg syncutils.MyWaitGroup
	wg.Go(func() {
		Polling(ctx, db, token, client)
	})

	sig := <-sigChan
	slog.Warn("Received signal", "signal", sig)
	cancel()
	wg.Wait()
}

func Polling(ctx context.Context, db *gorm.DB, token string, client pb.ProcessingServiceClient) {
	var offset int64 = 0
	for {
		// TODO: передивитися може щось подібне зробити десь ще
		select {
		case <-ctx.Done():
			return
		default:
		}

		updates, err := GetUpdates(ctx, token, offset)
		if err != nil {
			blog.Error("Trying to get updates", "error", err)
			time.Sleep(2 * time.Second)
			continue
		}

		for _, update := range updates {
			if err := HandleUpdate(ctx, db, client, update.Message, token); err != nil {
				blog.Error("Handling update", "error", err)
			}
			offset = update.UpdateID + 1
		}
	}
}
