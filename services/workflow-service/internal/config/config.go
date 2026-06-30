package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Log      LogConfig
}

type ServerConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type LogConfig struct {
	Level string
	JSON  bool
}

func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/app")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Defaults
	viper.SetDefault("SERVER_PORT", 8083)
	viper.SetDefault("SERVER_READ_TIMEOUT", "15s")
	viper.SetDefault("SERVER_WRITE_TIMEOUT", "15s")
	viper.SetDefault("SERVER_SHUTDOWN_TIMEOUT", "30s")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 25)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 10)
	viper.SetDefault("DB_CONN_MAX_LIFETIME", "5m")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_JSON", true)

	_ = viper.ReadInConfig()

	readTimeout, err := time.ParseDuration(viper.GetString("SERVER_READ_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("parsing SERVER_READ_TIMEOUT: %w", err)
	}
	writeTimeout, err := time.ParseDuration(viper.GetString("SERVER_WRITE_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("parsing SERVER_WRITE_TIMEOUT: %w", err)
	}
	shutdownTimeout, err := time.ParseDuration(viper.GetString("SERVER_SHUTDOWN_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("parsing SERVER_SHUTDOWN_TIMEOUT: %w", err)
	}
	connMaxLifetime, err := time.ParseDuration(viper.GetString("DB_CONN_MAX_LIFETIME"))
	if err != nil {
		return nil, fmt.Errorf("parsing DB_CONN_MAX_LIFETIME: %w", err)
	}

	dsn := viper.GetString("DATABASE_URL")
	if dsn == "" {
		host := viper.GetString("DB_HOST")
		port := viper.GetString("DB_PORT")
		user := viper.GetString("DB_USER")
		pass := viper.GetString("DB_PASSWORD")
		name := viper.GetString("DB_NAME")
		if host == "" {
			host = "localhost"
		}
		if port == "" {
			port = "3306"
		}
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC", user, pass, host, port, name)
	}

	return &Config{
		Server: ServerConfig{
			Port:            viper.GetInt("SERVER_PORT"),
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			ShutdownTimeout: shutdownTimeout,
		},
		Database: DatabaseConfig{
			DSN:             dsn,
			MaxOpenConns:    viper.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns:    viper.GetInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: connMaxLifetime,
		},
		Log: LogConfig{
			Level: viper.GetString("LOG_LEVEL"),
			JSON:  viper.GetBool("LOG_JSON"),
		},
	}, nil
}
