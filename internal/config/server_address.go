package config

import (
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"
)

const (
	DefaultServerAddress = "127.0.0.1"
	DefaultServerPort    = 8080
)

// ServerAddress represents the host and port information for a server.
//
// This struct is used to manage the server's address configuration, including
// the host name or IP address and the port number. It can be initialized with
// default values and can also be populated from environment variables.
type ServerAddress struct {
	Address string `env:"ADDRESS"`
	Host    string
	Port    int
}

// NewServerAddress creates and returns a new instance of the ServerAddress struct
// initialized with default values.
//
// The default host is set to "localhost" and the default port is set to 8080.
// This function is useful for quickly obtaining a ServerAddress instance
// without needing to specify the host and port manually.
//
// Returns:
// - Pointer to the newly created ServerAddress instance.
func NewServerAddress() *ServerAddress {
	return &ServerAddress{
		Host: DefaultServerAddress,
		Port: DefaultServerPort,
	}
}

// String returns the string representation of the ServerAddress in the format "host:port".
//
// Returns:
//   - The server address string formatted as "host:port".
func (s *ServerAddress) String() string {
	return s.Host + ":" + strconv.Itoa(s.Port)
}

// Set updates the Host and Port fields of the ServerAddress struct based on the provided
// address string in the format "host:port".
//
// Parameters:
//   - flagRunAddr: The address string to set, in the format "host:port".
//
// Returns:
//   - An error if the input format is invalid or if the port cannot be converted to an integer.
func (s *ServerAddress) Set(flagRunAddr string) error {
	splittedAddress := strings.Split(flagRunAddr, ":")

	if len(splittedAddress) != 2 {
		return errors.New("need address in a form host:port")
	}

	port, err := strconv.Atoi(splittedAddress[1])

	if err != nil {
		return err
	}

	s.Host = splittedAddress[0]
	s.Port = port

	return nil
}

// UnmarshalJSON unmarshals a JSON object into a ServerAddress struct.
// It expects the JSON to contain an "address" field in the format "host:port".
//
// Parameters:
//   - data: A byte slice containing the JSON data to unmarshal.
//
// Returns:
//   - An error if the input format is invalid or if the port cannot be converted to an integer.
func (s *ServerAddress) UnmarshalJSON(data []byte) error {
	address := strings.Trim(string(data), "\"")
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	s.Host = host
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return err
	}
	s.Port = port
	return nil
}

// MarshalJSON marshals the ServerAddress struct into a JSON string
// in the format "host:port".
func (s *ServerAddress) MarshalJSON() ([]byte, error) {
	address := s.String()
	return json.Marshal(address)
}
