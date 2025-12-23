package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	RPC               string
	USDC_TOKEN        string
	BatchSize         uint64
	Confirmations     uint64
	ExchangeAddresses map[string]bool
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
	}
}

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		println("Warning: .env file not found or could not be loaded")
	}
}
