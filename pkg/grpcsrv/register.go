package grpcsrv

import (
	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// StdRegister registers reflection and prometheus services.
func StdRegister(s *grpc.Server) {
	reflection.Register(s)
	grpcprometheus.EnableHandlingTimeHistogram(
		grpcprometheus.WithHistogramBuckets([]float64{0.02, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.8, 1, 1.2, 1.5, 2, 4, 8}),
	)
}
