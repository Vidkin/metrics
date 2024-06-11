package serverAddress

import (
	"errors"
	"strconv"
	"strings"
)

const (
	DefaultServerAddress = "localhost"
	DefaultServerPort    = 8080
)

type ServerAddress struct {
	Host    string
	Port    int
	Address string `env:"ADDRESS"`
}

func New() *ServerAddress {
	return &ServerAddress{
		Host: DefaultServerAddress,
		Port: DefaultServerPort,
	}
}

func (s *ServerAddress) String() string {
	return s.Host + ":" + strconv.Itoa(s.Port)
}

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
