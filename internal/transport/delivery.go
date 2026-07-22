package transport

import "google.golang.org/grpc"

type Server struct {
    srv *grpc.Server
}

func NewServer() *Server {
    return &Server{
        
    }
}