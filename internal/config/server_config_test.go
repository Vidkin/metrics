package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerConfig_LoadJSONConfig_Valid(t *testing.T) {
	jsonConfig := `{
		"address": "192.168.1.1:9090",
		"trusted_subnet": "192.168.1.0/24",
		"database_dsn": "user:password@tcp(localhost:3306)/dbname"
	}`
	file, err := os.CreateTemp("", "configServer.json")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	_, err = file.WriteString(jsonConfig)
	require.NoError(t, err)
	file.Close()

	config := &ServerConfig{ServerAddress: &ServerAddress{}}
	err = config.loadJSONConfig(file.Name())
	require.NoError(t, err)

	assert.Equal(t, "192.168.1.1:9090", config.ServerAddress.String())
	assert.Equal(t, "192.168.1.0/24", config.TrustedSubnet)
	assert.Equal(t, "user:password@tcp(localhost:3306)/dbname", config.DatabaseDSN)
}

func TestNewServerConfig(t *testing.T) {
	os.Setenv("TRUSTED_SUBNET", "192.168.1.0/24")
	os.Setenv("DATABASE_DSN", "user:password@tcp(localhost:3306)/dbname")
	defer os.Unsetenv("TRUSTED_SUBNET")
	defer os.Unsetenv("DATABASE_DSN")
	os.Args = os.Args[:1]

	config, err := NewServerConfig()
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.NotNil(t, config.ServerAddress)
	assert.Equal(t, 3, config.RetryCount)
	assert.Equal(t, "info", config.LogLevel)
	assert.Equal(t, "192.168.1.0/24", config.TrustedSubnet)
	assert.Equal(t, "user:password@tcp(localhost:3306)/dbname", config.DatabaseDSN)
}

func TestServerConfig_LoadJSONConfig_Invalid(t *testing.T) {
	config := &ServerConfig{}
	err := config.loadJSONConfig("invalid_path.json")
	assert.Error(t, err)
}
