package config

import (
	"flag"
	"os"
)

type AppConfig struct {
	ServerAddr                     string
	LogLevel                       string
	DatabaseURI                    string
	ContextTimeoutSec              int
	TokenSecretKey                 string
	TokenLifetimeSec               int
	AccrualSystemAddress           string
	AccrualSystemRequestTimeoutSec int
	AccrualMaxRequestsPerMinute    int
}

func ParseFlags() AppConfig {
	// Define defaults
	const (
		defaultServerAddress               = "localhost:8080"
		defaultLogLevel                    = "info"
		defaultDatabaseURI                 = "postgres://postgres:mysecretpassword@localhost:5432/postgres" //postgres://postgres:mysecretpassword@localhost:5432/postgres
		defaultContextTimeoutSec           = 20
		defaultTokenLifetimeSec            = 60 * 60 * 24 // 1 day
		defaultTokenSecret                 = "super-duper-secret"
		defaultAccrualSystemAddr           = "http://127.0.0.1:8081"
		defaultAccrualRequestTimeoutSec    = 30
		defaultAccrualMaxRequestsPerMinute = 60
	)

	// Initialize AppConfig with defaults
	config := AppConfig{
		ServerAddr:                     defaultServerAddress,
		LogLevel:                       defaultLogLevel,
		DatabaseURI:                    defaultDatabaseURI,
		ContextTimeoutSec:              defaultContextTimeoutSec,
		TokenLifetimeSec:               defaultTokenLifetimeSec,
		AccrualSystemAddress:           defaultAccrualSystemAddr,
		AccrualSystemRequestTimeoutSec: defaultAccrualRequestTimeoutSec,
		AccrualMaxRequestsPerMinute:    defaultAccrualMaxRequestsPerMinute,
		TokenSecretKey:                 defaultTokenSecret,
	}

	// Set flags
	flag.StringVar(&config.ServerAddr, "a", config.ServerAddr, "address and port to run server")
	flag.StringVar(&config.LogLevel, "ll", config.LogLevel, "logging level")
	flag.StringVar(&config.AccrualSystemAddress, "r", config.AccrualSystemAddress, "accrual system address")
	flag.StringVar(&config.DatabaseURI, "d", config.DatabaseURI, "database dsn")
	flag.Parse()

	// Override with environment variables if they exist
	if envVal := os.Getenv("RUN_ADDRESS"); envVal != "" {
		config.ServerAddr = envVal
	}
	if envVal := os.Getenv("LOG_LEVEL"); envVal != "" {
		config.LogLevel = envVal
	}
	if envVal := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envVal != "" {
		config.AccrualSystemAddress = envVal
	}
	if envVal := os.Getenv("DATABASE_URI"); envVal != "" {
		config.DatabaseURI = envVal
	}

	return config
}
