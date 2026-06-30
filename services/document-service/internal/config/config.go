package config

import "github.com/spf13/viper"

type Config struct {
	Port        string
	MongoURI    string
	MongoDBName string
	ServerEnv   string
	UploadDir   string
}

func Load() (*Config, error) {
	viper.SetDefault("DOCUMENT_SERVICE_PORT", "8084")
	viper.SetDefault("MONGO_URI", "mongodb://localhost:27017")
	viper.SetDefault("MONGO_DB_NAME", "documents_db")
	viper.SetDefault("SERVER_ENV", "development")
	viper.SetDefault("UPLOAD_DIR", "/tmp/opsnexus-uploads")
	viper.AutomaticEnv()

	return &Config{
		Port:        viper.GetString("DOCUMENT_SERVICE_PORT"),
		MongoURI:    viper.GetString("MONGO_URI"),
		MongoDBName: viper.GetString("MONGO_DB_NAME"),
		ServerEnv:   viper.GetString("SERVER_ENV"),
		UploadDir:   viper.GetString("UPLOAD_DIR"),
	}, nil
}
