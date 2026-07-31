package processing

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
)

var selog = slog.With("service", "processing")

func Sending(ctx context.Context, client pb.DataServiceClient, payload chan []byte) {
	for {
		select {
		case <-ctx.Done():
			return
		case p := <-payload:
			req := &pb.User{
				UserData: p,
			}
			_ = gRPCsender(ctx, client, req)
		}
	}
}

func gRPCsender(ctx context.Context, client pb.DataServiceClient, req *pb.User) *pb.Ack {
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := client.SendUser(callCtx, req)
	if err != nil {
		selog.Error("Could not send user", "error", err)
		return nil
	}
	return res
}
