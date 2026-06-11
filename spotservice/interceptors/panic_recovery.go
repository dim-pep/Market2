package interceptors

import (
	"context"
	"runtime/debug"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryPanicRecoveryInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			panicValue := recover()
			if panicValue == nil {
				return
			}

			logger.Error(
				"panic recovered from gRPC handler",
				zap.String("request_id", RequestIDFromContext(ctx)),
				zap.String("grpc.method", info.FullMethod),
				zap.Any("panic", panicValue),
				zap.ByteString("stack", debug.Stack()),
			)

			err = status.Error(codes.Internal, "internal error")
		}()

		return handler(ctx, req)
	}
}
