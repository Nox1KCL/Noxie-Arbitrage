package main

import (
	"context"
	"log/slog"
	"net"

	"github.com/Nox1KCL/Arbitrage/internal/syncutils"
	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
	"google.golang.org/grpc"
)

var dlog = slog.With("service", "delivery")

type server struct {
	pb.UnimplementedDataServiceServer
}

func (s *server) SendUser(ctx context.Context, req *pb.User) (*pb.Ack, error) {
	// TODO: Функція яка буде обробляти дані юзера і відправляти вже у месенджер мб

	return &pb.Ack{
		Status:  true,
		Details: "",
	}, nil
}

func main() {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		dlog.Error("Could not listen on port 50051", "error", err)
		return
	}
	grpcServer := grpc.NewServer()
	pb.RegisterDataServiceServer(grpcServer, &server{})

	dlog.Info("grpc server started successfully")

	var serviceWg syncutils.MyWaitGroup
	serviceWg.Go(func() {
		if err := grpcServer.Serve(listener); err != nil {
			dlog.Error("grpc server error", "error", err)
		}
	})
	serviceWg.Wait()
}
