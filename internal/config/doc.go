// Package config provides configuration management for the agent, including
// parsing command-line flags and environment variables.
//
// agent_config.go defines the `AgentConfig` struct, which holds configuration
// settings for the agent app, such as server address, report and poll intervals,
// rate limits, and logging levels. The package also provides functionality
// to initialize and parse these configurations from command-line flags and
// environment variables.
//
// server_address.go defines the `ServerAddress` struct, which holds the host and port
// information for a server. It provides functionality to initialize a server address
// with default values, convert the address to a string representation, and set the
// address from a formatted string.
//
// server_config.go defines the `ServerConfig` struct, which holds various configuration options
// for a server app, including server address, storage settings, logging preferences, and more.
// It supports loading configuration values from both command-line flags and environment variables.
//
// interval.go defines the Interval type, which represents a time interval
// in seconds. It includes a custom JSON unmarshalling method to parse
// interval strings that are expected to have a suffix of "s" (for seconds).
package config
