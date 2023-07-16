package grpcsrv

import (
	"context"

	"google.golang.org/grpc"
)

func UnaryReflectionFilter(exclude []string, in grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		for _, path := range exclude {
			if info.FullMethod == path {
				return handler(ctx, req)
			}
		}
		return in(ctx, req, info, handler)
	}
}

func StreamReflectionFilter(exclude []string, in grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		for _, path := range exclude {
			if info.FullMethod == path {
				return handler(srv, ss)
			}
		}
		return in(srv, ss, info, handler)
	}
}
