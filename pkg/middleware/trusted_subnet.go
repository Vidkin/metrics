package middleware

import (
	"net"
	"net/http"

	"github.com/Vidkin/metrics/internal/logger"
)

// TrustedSubnet is an HTTP middleware function that restricts access to
// incoming requests based on the client's IP address, allowing only
// requests from a specified subnet.
//
// Parameters:
//   - subnet: A string representing the CIDR notation of the trusted subnet.
//     Only clients with IP addresses within this subnet will be allowed to
//     access the next handler in the chain.
//
// Returns:
//   - A function that takes an http.Handler and returns a new http.Handler
//     that includes the subnet validation logic.
func TrustedSubnet(subnet string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, IPNet, err := net.ParseCIDR(subnet)
			if err != nil {
				logger.Log.Error("error parse subnet")
				w.WriteHeader(http.StatusForbidden)
				return
			}
			ipStr := r.Header.Get("X-Real-IP")
			ip := net.ParseIP(ipStr)
			if ip == nil {
				logger.Log.Error("error parse ip")
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if IPNet.Contains(ip) {
				next.ServeHTTP(w, r)
			}
			logger.Log.Error("error check trusted subnet")
			w.WriteHeader(http.StatusForbidden)
		})
	}
}
