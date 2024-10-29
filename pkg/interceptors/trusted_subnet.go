package interceptors

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/Vidkin/metrics/internal/logger"
)

func TrustedSubnetInterceptor(subnet string) func(context.Context, interface{}, *grpc.UnaryServerInfo, grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if subnet == "" {
			return handler(ctx, req)
		}

		_, IPNet, err := net.ParseCIDR(subnet)
		if err != nil {
			logger.Log.Error("error parse subnet")
			return nil, status.Error(codes.PermissionDenied, "error parse subnet")
		}
		var ipStr string
		if p, ok := peer.FromContext(ctx); ok {
			if addr, ok := p.Addr.(*net.TCPAddr); ok {
				ipStr = addr.IP.String()
			} else {
				logger.Log.Error("error get client ip address")
				return nil, status.Error(codes.PermissionDenied, "error get client ip address")
			}
		} else {
			logger.Log.Error("error get client ip address")
			return nil, status.Error(codes.PermissionDenied, "error get client ip address")
		}

		ip := net.ParseIP(ipStr)
		if ip == nil {
			logger.Log.Error("error parse ip")
			return nil, status.Error(codes.PermissionDenied, "error parse ip")
		}
		if IPNet.Contains(ip) {
			return handler(ctx, req)
		}
		logger.Log.Error("error check trusted subnet")
		return nil, status.Error(codes.PermissionDenied, "error check trusted subnet")
	}
}
