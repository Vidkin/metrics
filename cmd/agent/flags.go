package main

import (
	"errors"
	"strconv"
	"strings"
)

type ServerAddress struct {
	Host           string
	Port           int
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	Address        string `env:"ADDRESS"`
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
