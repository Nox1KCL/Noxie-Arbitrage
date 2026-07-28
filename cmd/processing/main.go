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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// TODO: Реалізувати функцію для пакування юзера і т.п
	// payload := []byte("... тут знаходиться ваш великий об'єм готових даних для користувача ...")
	// req := &pb.User{
	// 	UserData: payload,
	// }

	req := &pb.User{}
	res, err := client.SendUser(ctx, req)
	if err != nil {
		plog.Error("Could not send user", "error", err)
		return
	}

	plog.Info("User sent successfully", "status", res.Status, "details", res.Details)
}
