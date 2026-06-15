package interceptors

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const metadataUserRolesHeader = "x-user-roles"

func UnaryLoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		grpcCode := status.Code(err)

		fields := make([]zap.Field, 0, 8)

		if requestID := RequestIDFromContext(ctx); requestID != "" {
			fields = append(fields, zap.String("request_id", requestID))
		}

		fields = append(
			fields,
			zap.String("grpc.method", info.FullMethod),
			zap.Duration("grpc.duration", duration),
			zap.Int64("grpc.duration_ms", duration.Milliseconds()),
			zap.String("grpc.code", grpcCode.String()),
		)

		if roles, ok := UserRolesFromMetadata(ctx); ok {
			fields = append(fields, zap.Strings("user_roles", roles))
		}

		if clientPeer, ok := peer.FromContext(ctx); ok && clientPeer.Addr != nil {
			fields = append(fields, zap.String("grpc.peer", clientPeer.Addr.String()))
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
		}

		switch grpcCode {
		case codes.OK:
			logger.Info("gRPC request completed", fields...)

		case codes.Canceled, codes.DeadlineExceeded:
			logger.Warn("gRPC request cancelled or timed out", fields...)

		case codes.InvalidArgument, codes.NotFound, codes.FailedPrecondition, codes.PermissionDenied, codes.AlreadyExists, codes.Unauthenticated:
			logger.Info("gRPC client error", fields...)

		case codes.Unavailable, codes.ResourceExhausted, codes.Aborted:
			logger.Warn("gRPC transient failure", fields...)

		default:
			logger.Error("gRPC request failed", fields...)
		}

		return resp, err
	}
}

func UserRolesFromMetadata(ctx context.Context) ([]string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, false
	}

	values := md.Get(metadataUserRolesHeader)
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return nil, false
	}

	rawRoles := strings.Split(values[0], ",")
	roles := make([]string, 0, len(rawRoles))

	for _, role := range rawRoles {
		role = strings.TrimSpace(role)
		if role != "" {
			roles = append(roles, role)
		}
	}

	return roles, len(roles) > 0
}
