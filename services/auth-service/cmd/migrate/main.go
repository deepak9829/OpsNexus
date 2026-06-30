package main

import (
	"fmt"
	"os"

	mysqladapter "github.com/opsnexus/auth-service/internal/adapters/mysql"
	"github.com/opsnexus/auth-service/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	db, err := mysqladapter.NewConnection(cfg.Database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Running database migrations...")
	if err := mysqladapter.AutoMigrate(db); err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Migrations completed successfully.")
}
