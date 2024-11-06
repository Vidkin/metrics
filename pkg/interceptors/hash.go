package interceptors

import (
	"bytes"
	"context"
	"encoding/base64"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/pkg/hash"
)

func HashInterceptor(key string) func(context.Context, interface{}, *grpc.UnaryServerInfo, grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if key == "" {
			return handler(ctx, req)
		}

		var hEnc string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			values := md.Get("HashSHA256")
			if len(values) > 0 {
				hEnc = values[0]
			}
		}
		if len(hEnc) == 0 {
			return nil, status.Error(codes.InvalidArgument, "missing hash")
		}

		hashA, err := base64.StdEncoding.DecodeString(hEnc)
		if err != nil {
			logger.Log.Error("error decode hash from base64 string", zap.Error(err))
			return nil, status.Error(codes.Internal, "missing hash")
		}

		var data []byte
		if msg, ok := req.(proto.Message); ok {
			data, err = proto.Marshal(msg)
			if err != nil {
				logger.Log.Error("failed to marshal request: %v", zap.Error(err))
				return nil, status.Errorf(codes.Internal, "failed to marshal request")
			}
		} else {
			return nil, status.Errorf(codes.Internal, "failed to get proto.Message")
		}

		hashB := hash.GetHashSHA256(key, data)
		if !bytes.Equal(hashA, hashB) {
			logger.Log.Error("hashes don't match")
			return nil, status.Errorf(codes.InvalidArgument, "hashes don't match")
		}

		return handler(ctx, req)
	}
}
