package middlewares

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		code := status.Code(err)

		level := slog.LevelInfo
		switch code {
		case codes.OK, codes.NotFound, codes.Canceled, codes.AlreadyExists, codes.InvalidArgument, codes.Unauthenticated:
			level = slog.LevelInfo
		case codes.DeadlineExceeded, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unavailable:
			level = slog.LevelWarn
		case codes.Unknown, codes.Internal, codes.Unimplemented, codes.DataLoss:
			level = slog.LevelError
		}

		slog.LogAttrs(ctx, level, "gRPC request",
			slog.String("method", info.FullMethod),
			slog.String("status", code.String()),
			slog.Duration("duration", duration),
		)

		return resp, err
	}
}
