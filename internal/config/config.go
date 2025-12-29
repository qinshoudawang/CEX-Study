package config

import (
	"dex-indexer/internal/middleware/db"
	redisclient "dex-indexer/internal/middleware/redis"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	RPC               string
	USDC_TOKEN        string
	BatchSize         uint64 // number of blocks to process in one batch
	ExchangeAddresses map[string]bool
	DBConfig          *db.Config
	RedisConfig       *redisclient.Config
}

// Load loads the configuration
func Load() *Config {
	return &Config{
		RPC:        os.Getenv("ETH_RPC_URL"),
		USDC_TOKEN: os.Getenv("USDC_TOKEN"),
		BatchSize:  10,
		ExchangeAddresses: map[string]bool{
			os.Getenv("EXCHANGE_ADDRESS"): true,
		},
		DBConfig: &db.Config{
			Host:     os.Getenv("DB_HOST"),
			Port:     parseEnvAsInt("DB_PORT", 5432),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
			SSLMode:  os.Getenv("DB_SSLMODE"),
		},
		RedisConfig: &redisclient.Config{
			Addr:     os.Getenv("REDIS_ADDR"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       parseEnvAsInt("REDIS_DB", 0),
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
