package grpcsrv

import (
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

type GrpcServerWorker struct {
	name       string
	grpcServer *grpc.Server
	bind       string
}

func NewGrpcServerWorker(name string, grpcServer *grpc.Server, bind string) *GrpcServerWorker {
	return &GrpcServerWorker{name: name, grpcServer: grpcServer, bind: bind}
}

func (g *GrpcServerWorker) Start() error {
	log.Info().
		Msgf("start grpc server worker: %s %s", g.bind, g.name)

	return ListenAndServe(g.grpcServer, g.bind)
}

func (g *GrpcServerWorker) Stop() error {
	defer log.Info().
		Msgf("%s grpc server stopped", g.name)

	g.grpcServer.GracefulStop()

	return nil
}
