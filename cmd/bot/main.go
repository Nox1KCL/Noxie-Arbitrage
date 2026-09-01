package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Nox1KCL/Arbitrage/internal/bot"
	"github.com/Nox1KCL/Arbitrage/internal/config"
	"github.com/Nox1KCL/Arbitrage/internal/database"
	"github.com/Nox1KCL/Arbitrage/internal/database/models"
	telemetry "github.com/Nox1KCL/Arbitrage/internal/observer"
	"github.com/Nox1KCL/Arbitrage/internal/syncutils"
	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var blog = slog.With("service", "bot")

func main() {
	syncutils.LoadEnv()

	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		blog.ErrorContext(ctx, "TELEGRAM_BOT_TOKEN is not set")
		os.Exit(1)
	}

	conn, err := grpc.NewClient("processing:50050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		blog.ErrorContext(ctx, "Could not connect to client", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewProcessingServiceClient(conn)
	blog.InfoContext(ctx, "grpc server connection finished successfully")

	db, err := database.Connect()
	if err != nil {
		blog.ErrorContext(ctx, "Could not connect to db", "error", err)
		os.Exit(1)
	}
	if err := db.AutoMigrate(&models.Subscription{}); err != nil {
		blog.ErrorContext(ctx, "Could not create a table", "error", err)
		os.Exit(1)
	}
	blog.Info("db init finished successfully")

	cfg, err := config.GetConfig("config.toml")
	if err != nil {
		blog.ErrorContext(ctx, "Could not load config", "error", err)
		os.Exit(1)
	}

	shutdown, observer, err := telemetry.NewTelemetry(&cfg.LumberConfig)
	if err != nil {
		blog.ErrorContext(ctx, "Could not get observer", "error", err)
		os.Exit(1)
	}
	defer shutdown(ctx)
	blog.InfoContext(ctx, "telemetry init finished successfully")

	service, err := bot.NewBotService(observer, db, token, client)
	if err != nil {
		blog.ErrorContext(ctx, "Could not get bot service", "error", err)
		os.Exit(1)
	}
	_, span := service.Observer.Tracer.Start(ctx, "BotInit")
	blog.InfoContext(ctx, "botService init finished successfully")

	var wg syncutils.MyWaitGroup
	errChan := make(chan error, 1)
	wg.Go(func() {
		errChan <- service.Polling(ctx)
		err := <-errChan
		if err != nil {
			blog.InfoContext(ctx, "gracefully shutdown polling process")
		}
	})

	sig := <-sigChan
	blog.WarnContext(ctx, "Received signal", "signal", sig)
	cancel()
	wg.Wait()
	span.AddEvent("received signal, gracefully shutdown..")
	span.SetAttributes(
		attribute.String("os.signal.name", sig.String()),
	)
	span.End()
}
