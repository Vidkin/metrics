package interceptors

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Vidkin/metrics/internal/logger"
)

func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	startTime := time.Now()
	itf, err := handler(ctx, req)
	duration := time.Since(startTime)

	var respStatus string
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			respStatus = st.Code().String()
		} else {
			respStatus = err.Error()
		}
	} else {
		respStatus = codes.OK.String()
	}
	logger.Log.Info(
		"Request data",
		zap.String("method", info.FullMethod),
		zap.Duration("duration", duration),
	)
	logger.Log.Info(
		"Response data",
		zap.String("status", respStatus),
	)
	return itf, err
}
