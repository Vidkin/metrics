package config

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentConfig_loadJSONConfig_Valid(t *testing.T) {
	tempFile, err := os.CreateTemp("", "configAgent.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	testConfig := AgentConfig{
		ServerAddress:  NewServerAddress(),
		CryptoKey:      "testCryptoKey",
		Key:            "testKey",
		ReportInterval: 15,
		PollInterval:   5,
		RateLimit:      10,
		UseGRPC:        false,
	}
	data, err := json.Marshal(testConfig)
	require.NoError(t, err)
	_, err = tempFile.Write(data)
	require.NoError(t, err)

	// Загружаем конфигурацию из файла
	config := &AgentConfig{
		ServerAddress: NewServerAddress(),
	}
	err = config.loadJSONConfig(tempFile.Name())
	require.NoError(t, err)

	assert.Equal(t, "testCryptoKey", config.CryptoKey)
	assert.Equal(t, "testKey", config.Key)
	assert.Equal(t, 15, int(config.ReportInterval))
	assert.Equal(t, 5, int(config.PollInterval))
	assert.Equal(t, 10, config.RateLimit)
	assert.False(t, config.UseGRPC)
}

func TestNewAgentConfig(t *testing.T) {
	// Сохраняем оригинальные os.Args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }() // Восстанавливаем после теста

	// Устанавливаем флаги для тестирования
	os.Args = []string{"cmd", "-a", "192.168.1.1:8080", "-c", "config.json", "-r", "15", "-p", "5", "-l", "10", "-k", "testKey", "-crypto-key", "testCryptoKey", "-g", "true"}

	// Вызываем функцию NewAgentConfig
	config, err := NewAgentConfig()
	require.NoError(t, err)

	// Проверяем значения конфигурации
	assert.NotNil(t, config)
	assert.Equal(t, "192.168.1.1:8080", config.ServerAddress.String())
	assert.Equal(t, "config.json", config.ConfigPath)
	assert.Equal(t, 15, int(config.ReportInterval))
	assert.Equal(t, 5, int(config.PollInterval))
	assert.Equal(t, 10, config.RateLimit)
	assert.Equal(t, "testKey", config.Key)
	assert.True(t, config.UseGRPC)
	assert.Equal(t, "testCryptoKey", config.CryptoKey)
	assert.Equal(t, "info", config.LogLevel)
}

func TestAgentConfig_loadJSONConfig_InvalidFile(t *testing.T) {
	config := &AgentConfig{
		ServerAddress: NewServerAddress(),
	}
	err := config.loadJSONConfig("invalid_path.json")
	assert.Error(t, err)
}

func TestAgentConfig_loadJSONConfig_InvalidJSON(t *testing.T) {
	// Создаем временный файл с некорректным JSON
	tempFile, err := os.CreateTemp("", "invalid_config.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write([]byte(`{invalid json`))
	require.NoError(t, err)

	config := &AgentConfig{
		ServerAddress: NewServerAddress(),
	}
	err = config.loadJSONConfig(tempFile.Name())
	assert.Error(t, err)
}
