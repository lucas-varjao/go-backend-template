// backend/internal/config/config.go

package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type DatabaseConfig struct {
	DSN string `mapstructure:"dsn"`
}

type JWTConfig struct {
	SecretKey        string        `mapstructure:"secret-key"`
	AccessTokenTTL   time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL  time.Duration `mapstructure:"refresh_token_ttl"`
	PasswordResetTTL time.Duration `mapstructure:"password_reset_ttl"`
	Issuer           string        `mapstructure:"issuer"`
}

// EmailConfig contém configurações para envio de email
type EmailConfig struct {
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	SMTPUsername string `mapstructure:"smtp_username"`
	SMTPPassword string `mapstructure:"smtp_password"`
	FromEmail    string `mapstructure:"from_email"`
	FromName     string `mapstructure:"from_name"`
	ResetURL     string `mapstructure:"reset_url"`
}

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Email    EmailConfig    `mapstructure:"email"`
}

var cfg *Config

func LoadConfig() (*Config, error) {
	viper.SetConfigName("app")
	viper.SetConfigType("yml")
	viper.AddConfigPath("./configs")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("falha ao ler o arquivo de configuração: %w", err)
	}

	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("falha ao carregar as configurações: %w", err)
	}

	return cfg, nil

}

func GetConfig() *Config {
	return cfg
}
