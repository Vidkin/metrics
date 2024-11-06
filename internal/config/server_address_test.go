package config

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServerAddress(t *testing.T) {
	address := NewServerAddress()
	assert.NotNil(t, address)
	assert.Equal(t, DefaultServerAddress, address.Host)
	assert.Equal(t, DefaultServerPort, address.Port)
}

func TestServerAddress_String(t *testing.T) {
	address := NewServerAddress()
	expected := DefaultServerAddress + ":" + strconv.Itoa(DefaultServerPort)
	assert.Equal(t, expected, address.String())
}

func TestServerAddress_Set_Valid(t *testing.T) {
	address := NewServerAddress()
	err := address.Set("192.168.1.1:9090")
	require.NoError(t, err)
	assert.Equal(t, "192.168.1.1", address.Host)
	assert.Equal(t, 9090, address.Port)
}

func TestServerAddress_Set_InvalidFormat(t *testing.T) {
	address := NewServerAddress()
	err := address.Set("invalidAddress")
	assert.Error(t, err)
	assert.Equal(t, "need address in a form host:port", err.Error())
}

func TestServerAddress_Set_InvalidPort(t *testing.T) {
	address := NewServerAddress()
	err := address.Set("192.168.1.1:invalidPort")
	assert.Error(t, err)
}

func TestServerAddress_UnmarshalJSON_Valid(t *testing.T) {
	address := NewServerAddress()
	data := []byte(`"192.168.1.1:9090"`)
	err := address.UnmarshalJSON(data)
	require.NoError(t, err)
	assert.Equal(t, "192.168.1.1", address.Host)
	assert.Equal(t, 9090, address.Port)
}

func TestServerAddress_UnmarshalJSON_InvalidFormat(t *testing.T) {
	address := NewServerAddress()
	data := []byte(`"invalidAddress"`)
	err := address.UnmarshalJSON(data)
	assert.Error(t, err)
}

func TestServerAddress_UnmarshalJSON_InvalidPort(t *testing.T) {
	address := NewServerAddress()
	data := []byte(`"192.168.1.1:invalidPort"`)
	err := address.UnmarshalJSON(data)
	assert.Error(t, err)
}

func TestServerAddress_MarshalJSON_Valid(t *testing.T) {
	address := &ServerAddress{
		Host: "192.168.1.1",
		Port: 9090,
	}
	expected := `"192.168.1.1:9090"`
	data, err := address.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, expected, string(data))
}

func TestServerAddress_MarshalJSON_DefaultValues(t *testing.T) {
	address := NewServerAddress()
	expected := `"127.0.0.1:8080"`
	data, err := address.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, expected, string(data))
}
