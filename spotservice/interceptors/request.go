package interceptors

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const metadataRequestIDHeader = "x-request-id"

type requestIDContextKey struct{}

var requestIDKey = requestIDContextKey{}

func UnaryRequestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		requestID := requestIDFromIncomingMetadata(ctx)
		if requestID == "" {
			requestID = uuid.NewString()
		}

		ctx = context.WithValue(ctx, requestIDKey, requestID)

		return handler(ctx, req)
	}
}

func RequestIDFromContext(ctx context.Context) string {
	value, ok := ctx.Value(requestIDKey).(string)
	if !ok {
		return ""
	}

	return value
}

func requestIDFromIncomingMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	values := md.Get(metadataRequestIDHeader)
	if len(values) == 0 {
		return ""
	}

	return values[0]
}
