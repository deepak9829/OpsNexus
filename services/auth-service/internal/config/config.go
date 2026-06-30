package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
}

type ServerConfig struct {
	Port string
	Env  string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type JWTConfig struct {
	Secret           string
	AccessTTLMinutes int
	RefreshTTLHours  int
}

func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("AUTH_SERVICE_PORT", "8081")
	viper.SetDefault("SERVER_ENV", "development")
	viper.SetDefault("MYSQL_HOST", "localhost")
	viper.SetDefault("MYSQL_PORT", "3306")
	viper.SetDefault("AUTH_DB_NAME", "auth_db")
	viper.SetDefault("JWT_ACCESS_TTL_MINUTES", 15)
	viper.SetDefault("JWT_REFRESH_TTL_HOURS", 168)

	_ = viper.ReadInConfig()

	return &Config{
		Server: ServerConfig{
			Port: viper.GetString("AUTH_SERVICE_PORT"),
			Env:  viper.GetString("SERVER_ENV"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("MYSQL_HOST"),
			Port:     viper.GetString("MYSQL_PORT"),
			User:     viper.GetString("AUTH_DB_USER"),
			Password: viper.GetString("AUTH_DB_PASSWORD"),
			Name:     viper.GetString("AUTH_DB_NAME"),
		},
		JWT: JWTConfig{
			Secret:           viper.GetString("JWT_SECRET"),
			AccessTTLMinutes: viper.GetInt("JWT_ACCESS_TTL_MINUTES"),
			RefreshTTLHours:  viper.GetInt("JWT_REFRESH_TTL_HOURS"),
		},
	}, nil
}
