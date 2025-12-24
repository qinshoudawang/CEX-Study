package config

import (
	"dex-indexer/internal/db"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	RPC               string
	USDC_TOKEN        string
	BatchSize         uint64
	Confirmations     uint64
	ExchangeAddresses map[string]bool
	DBConfig          *db.Config
}

// Load loads the configuration
func Load() *Config {
	return &Config{
		RPC:           os.Getenv("ETH_RPC_URL"),
		USDC_TOKEN:    os.Getenv("USDC_TOKEN"),
		BatchSize:     10,
		Confirmations: 10,
		ExchangeAddresses: map[string]bool{
			strings.ToLower(os.Getenv("EXCHANGE_ADDRESS")): true,
		},
		DBConfig: &db.Config{
			Host:     os.Getenv("DB_HOST"),
			Port:     parseEnvAsInt("DB_PORT", 5432),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
			SSLMode:  os.Getenv("DB_SSLMODE"),
		},
	}
}

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		println("Warning: .env file not found or could not be loaded")
	}
}

// parseEnvAsInt parses an environment variable as an integer, returning a default value if parsing fails.
func parseEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
