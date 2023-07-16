package grpcsrv

import (
	"net"

	"google.golang.org/grpc"
)

func ListenAndServe(s *grpc.Server, bind string) error {
	listener, err := net.Listen("tcp", bind)
	if err != nil {
		return err
	}

	if err = s.Serve(listener); err != nil {
		return err
	}

	return nil
}
