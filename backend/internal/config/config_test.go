// backend/internal/config/config_test.go

package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func setupTestConfig(t *testing.T) func() {
	// Create a temporary config file for testing
	configContent := `
server:
  port: 8080
database:
  dsn: "test.db"
jwt:
  secret-key: "test-secret-key"
  access_token_ttl: 15m
  refresh_token_ttl: 24h
  issuer: "gosveltekit-test"
`
	err := os.MkdirAll("./configs", 0755)
	assert.NoError(t, err)

	err = os.WriteFile("./configs/app.yml", []byte(configContent), 0644)
	assert.NoError(t, err)

	// Return cleanup function
	return func() {
		os.RemoveAll("./configs")
		// Reset viper to avoid interference between tests
		viper.Reset()
		cfg = nil
	}
}

func TestLoadConfig(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	config, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Verify loaded values
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, "test.db", config.Database.DSN)
	assert.Equal(t, "test-secret-key", config.JWT.SecretKey)
	assert.Equal(t, 15*time.Minute, config.JWT.AccessTokenTTL)
	assert.Equal(t, 24*time.Hour, config.JWT.RefreshTokenTTL)
	assert.Equal(t, "gosveltekit-test", config.JWT.Issuer)
}

func TestLoadConfigError(t *testing.T) {
	// Reset viper to avoid interference from other tests
	viper.Reset()
	cfg = nil

	// Test with non-existent config file
	config, err := LoadConfig()
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestGetConfig(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// First, load the config
	_, err := LoadConfig()
	assert.NoError(t, err)

	// Then get it
	config := GetConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 8080, config.Server.Port)
}

func TestGetConfigBeforeLoad(t *testing.T) {
	// Reset viper and cfg
	viper.Reset()
	cfg = nil

	// GetConfig should return nil if LoadConfig hasn't been called
	config := GetConfig()
	assert.Nil(t, config)
}
