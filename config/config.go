package config

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config struct to hold the configuration
type Config struct {
	Port     string `envconfig:"PORT" default:"8080"`
	StoreDir string `envconfig:"STORE_DIR" default:"./store"`
}

// Load function to load the configuration from the environment variables
func Load() (Config, error) {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env file found")
	}

	var c Config
	err = envconfig.Process("", &c)
	if err != nil {
		return Config{}, fmt.Errorf("unable to get envconfig: %w", err)
	}

	return c, nil
}
