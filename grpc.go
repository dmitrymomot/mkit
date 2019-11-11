package mkit

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/dmitrymomot/mkit/logger"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

// Server struct
type Server struct {
	grpcServer *grpc.Server
	grpcPort   int
}

// GRPCServer ...
func (s *Server) GRPCServer(g *grpc.Server, port int) {
	s.grpcServer = g
	s.grpcPort = port
}

// Run server
func (s *Server) Run() error {
	// Listen interrupt signal from OS
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	// Context with the cancellation function
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Error group with custom context
	eg, ctx := errgroup.WithContext(ctx)

	// Run gRPC server
	eg.Go(func() error {
		addr := fmt.Sprintf(":%d", s.grpcPort)
		grpcLis, err := net.Listen("tcp", addr)
		if err != nil {
			return errors.Wrap(err, "grpc failed to listen")
		}
		logger.Info().Str("listen", addr).Msg("gRPC server started")
		if err := s.grpcServer.Serve(grpcLis); err != nil {
			return errors.Wrap(err, "grpc server")
		}
		return nil
	})

	// Wait for interrupt signal or context cancellation
	select {
	case <-interrupt:
		break
	case <-ctx.Done():
		break
	}

	logger.Info().Msg("received shutdown signal")

	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}

	if err := eg.Wait(); err != nil {
		return errors.Wrap(err, "failed to wait goroutine group")
	}

	logger.Info().Msg("service stopped")

	return nil
}
