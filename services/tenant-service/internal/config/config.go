package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the tenant service.
type Config struct {
	Port      int    `mapstructure:"TENANT_SERVICE_PORT"`
	MySQLHost string `mapstructure:"MYSQL_HOST"`
	MySQLPort int    `mapstructure:"MYSQL_PORT"`
	DBUser    string `mapstructure:"TENANT_DB_USER"`
	DBPass    string `mapstructure:"TENANT_DB_PASSWORD"`
	DBName    string `mapstructure:"TENANT_DB_NAME"`
	ServerEnv string `mapstructure:"SERVER_ENV"`
}

// DSN returns the MySQL data source name.
func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUser, c.DBPass, c.MySQLHost, c.MySQLPort, c.DBName)
}

// Load reads configuration from environment variables (and optionally a .env file).
func Load() (*Config, error) {
	v := viper.New()

	v.SetDefault("TENANT_SERVICE_PORT", 8082)
	v.SetDefault("MYSQL_HOST", "localhost")
	v.SetDefault("MYSQL_PORT", 3306)
	v.SetDefault("TENANT_DB_USER", "root")
	v.SetDefault("TENANT_DB_PASSWORD", "")
	v.SetDefault("TENANT_DB_NAME", "tenant_db")
	v.SetDefault("SERVER_ENV", "development")

	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Best-effort .env load; ignore if not found.
	_ = v.ReadInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}
