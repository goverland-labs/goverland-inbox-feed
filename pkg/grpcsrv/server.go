package grpcsrv

import (
	"google.golang.org/grpc"
)

func NewGrpcServer() *grpc.Server {
	server := grpc.NewServer()
	StdRegister(server)

	return server
}
