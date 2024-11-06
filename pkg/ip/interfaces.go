// Package ip provides utilities for retrieving network interface information.
//
// This package includes functions to get the IP addresses of all active network
// interfaces on the local machine. It can be useful for applications that need
// to determine the local IP addresses for networking purposes, such as
// server applications or network diagnostics.
package ip

import (
	"net"

	"go.uber.org/zap"

	"github.com/Vidkin/metrics/internal/logger"
)

// GetMyInterfaces retrieves the IP addresses of all active network interfaces.
//
// This function scans all network interfaces on the local machine and returns
// a slice of strings containing the IPv4 addresses of those interfaces that are
// currently up (active).
//
// Returns:
//   - A slice of strings containing the IPv4 addresses of active network interfaces.
//   - An error if there was an issue retrieving the interfaces or their addresses.
//
// If an error occurs while fetching the interfaces or their addresses, it logs
// the error using the logger package and returns the error to the caller.
func GetMyInterfaces() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		logger.Log.Error("error getting interfaces", zap.Error(err))
		return nil, err
	}

	res := make([]string, 0, len(interfaces))
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			logger.Log.Error("error getting addresses", zap.Error(err))
			return nil, err
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				res = append(res, ipnet.IP.String())
			}
		}
	}
	return res, nil
}
