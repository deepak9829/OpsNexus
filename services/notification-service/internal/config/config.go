package config

import "github.com/spf13/viper"

type Config struct {
	Port                       string
	AWSEndpointURL             string
	AWSRegion                  string
	AWSAccessKeyID             string
	AWSSecretAccessKey         string
	DynamoDBNotificationsTable string
	DynamoDBAuditTable         string
	ServerEnv                  string
}

func Load() (*Config, error) {
	viper.SetDefault("NOTIFICATION_SERVICE_PORT", "8085")
	viper.SetDefault("AWS_ENDPOINT_URL", "")
	viper.SetDefault("AWS_REGION", "us-east-1")
	viper.SetDefault("AWS_ACCESS_KEY_ID", "")
	viper.SetDefault("AWS_SECRET_ACCESS_KEY", "")
	viper.SetDefault("DYNAMODB_NOTIFICATIONS_TABLE", "notifications")
	viper.SetDefault("DYNAMODB_AUDIT_TABLE", "audit_events")
	viper.SetDefault("SERVER_ENV", "development")
	viper.AutomaticEnv()

	return &Config{
		Port:                       viper.GetString("NOTIFICATION_SERVICE_PORT"),
		AWSEndpointURL:             viper.GetString("AWS_ENDPOINT_URL"),
		AWSRegion:                  viper.GetString("AWS_REGION"),
		AWSAccessKeyID:             viper.GetString("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey:         viper.GetString("AWS_SECRET_ACCESS_KEY"),
		DynamoDBNotificationsTable: viper.GetString("DYNAMODB_NOTIFICATIONS_TABLE"),
		DynamoDBAuditTable:         viper.GetString("DYNAMODB_AUDIT_TABLE"),
		ServerEnv:                  viper.GetString("SERVER_ENV"),
	}, nil
}
