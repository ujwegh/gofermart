package config

import (
	"flag"
	"os"
)

type AppConfig struct {
	ServerAddr        string
	LogLevel          string
	DatabaseDSN       string
	ContextTimeoutSec int
	TokenSecretKey    string
	TokenLifetimeSec  int
}

func ParseFlags() AppConfig {
	// Define defaults
	const (
		defaultServerAddress     = "localhost:8080"
		defaultLogLevel          = "info"
		defaultDatabaseDSN       = "" //postgres://postgres:mysecretpassword@localhost:5432/postgres
		defaultContextTimeoutSec = 5
		defaultTokenLifetimeSec  = 60 * 60 * 24 // 1 day
	)

	// Initialize AppConfig with defaults
	config := AppConfig{
		ServerAddr:        defaultServerAddress,
		LogLevel:          defaultLogLevel,
		DatabaseDSN:       defaultDatabaseDSN,
		ContextTimeoutSec: defaultContextTimeoutSec,
		TokenLifetimeSec:  defaultTokenLifetimeSec,
	}

	// Set flags
	flag.StringVar(&config.ServerAddr, "a", config.ServerAddr, "address and port to run server")
	flag.StringVar(&config.LogLevel, "ll", config.LogLevel, "logging level")
	flag.StringVar(&config.DatabaseDSN, "d", config.DatabaseDSN, "database dsn")
	flag.Parse()

	// Override with environment variables if they exist
	if envVal := os.Getenv("SERVER_ADDRESS"); envVal != "" {
		config.ServerAddr = envVal
	}
	if envVal := os.Getenv("LOG_LEVEL"); envVal != "" {
		config.LogLevel = envVal
	}
	if envVal := os.Getenv("DATABASE_DSN"); envVal != "" {
		config.DatabaseDSN = envVal
	}

	return config
}
