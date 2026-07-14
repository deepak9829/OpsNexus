package config

import (
	"fmt"
	"net/url"

	"github.com/spf13/viper"
)

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

	mongoURI := viper.GetString("MONGO_URI")

	// When individual DocDB credentials are provided, build the URI with
	// properly URL-encoded components so special characters in the password
	// (%, [, >, etc.) don't break the MongoDB driver's URI parser.
	host := viper.GetString("DOCDB_HOST")
	port := viper.GetString("DOCDB_PORT")
	user := viper.GetString("DOCDB_USER")
	pass := viper.GetString("DOCDB_PASSWORD")
	tlsCAFile := viper.GetString("DOCDB_TLS_CA_FILE")

	if host != "" && user != "" && pass != "" {
		if port == "" {
			port = "27017"
		}
		query := "tls=true&replicaSet=rs0&readPreference=secondaryPreferred&retryWrites=false"
		if tlsCAFile != "" {
			query += "&tlsCAFile=" + url.QueryEscape(tlsCAFile)
		}
		mongoURI = fmt.Sprintf("mongodb://%s:%s@%s:%s/?%s",
			url.QueryEscape(user),
			url.QueryEscape(pass),
			host,
			port,
			query,
		)
	}

	return &Config{
		Port:        viper.GetString("DOCUMENT_SERVICE_PORT"),
		MongoURI:    mongoURI,
		MongoDBName: viper.GetString("MONGO_DB_NAME"),
		ServerEnv:   viper.GetString("SERVER_ENV"),
		UploadDir:   viper.GetString("UPLOAD_DIR"),
	}, nil
}
