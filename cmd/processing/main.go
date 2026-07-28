package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Nox1KCL/Arbitrage/internal/broker"
	"github.com/Nox1KCL/Arbitrage/internal/processing"
	"github.com/Nox1KCL/Arbitrage/internal/syncutils"
	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
)

var plog = slog.With("service", "processing")

func main() {
	conn, err := grpc.NewClient("delivery-go:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		plog.Error("Could not connect to client", "error", err)
		return
	}
	defer conn.Close()

	client := pb.NewDataServiceClient(conn)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	payload := make(chan []byte)
	b, err := broker.NewBroker("parser.binance.ticker")
	if err != nil {
		plog.Error("Could not create broker", "error", err)
		return
	}

	msgs, err := b.Unload()
	if err != nil {
		plog.Error("Could not unload broker", "error", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	tickerChannel := make(chan processing.TickerForm)
	var serviceWg syncutils.MyWaitGroup
	serviceWg.Go(func() {
        processing.Scanner(ctx, msgs, tickerChannel)
	})

	serviceWg.Go(func() {
	    processing.Filter(ctx, tickerChannel, payload)
	})

	serviceWg.Go(func() {
        processing.Sending(ctx, client, payload)
	})

	sig := <-sigChan
	plog.Warn("Received signal", "signal", sig)

	cancel()
	serviceWg.Wait()

	// 1) gathering як горутина +
	// 2) filter як горутина +
	// 3) forming це вже запускаємо у fiter
	// 4) sender запускаємо коли з канала (який тут), приходить щось сюди
	// 5) sender теж запускається в горутині, і якщо payload щось приходить, запускає SendUser

}
