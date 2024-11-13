package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type RabbitMQConfig struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	User        string `mapstructure:"user"`
	Password    string `mapstructure:"password"`
	Queue       string `mapstructure:"queue"`
	QueueType   string `mapstructure:"queue_type"`
	RoutingKey  string `mapstructure:"routing_key"`
	SourceChain string `mapstructure:"source_chain"`
	Enabled     *bool  `mapstructure:"enabled"`
	StopHeight  *int64 `mapstructure:"stop_height"`
}

type EventBusConfig struct {
}

type Config struct {
	ConfigPath        string         `mapstructure:"config_path"`
	ChainEnv          string         `mapstructure:"chain_env"`
	ConnnectionString string         `mapstructure:"connection_string"` // Postgres db connection string
	ScalarMnemonic    string         `mapstructure:"scalar_mnemonic"`
	EvmPrivateKey     string         `mapstructure:"evm_private_key"`
	BtcPrivateKey     string         `mapstructure:"btc_private_key"`
	EventBus          EventBusConfig `mapstructure:"event_bus"`
}

var GlobalConfig *Config

func LoadEnv() error {
	// Tell Viper to read from environment
	viper.AutomaticEnv()

	// Add support for .env files
	viper.SetConfigName(".env.local") // name of config file (without extension)
	viper.SetConfigType("env")        // type of config file
	viper.AddConfigPath(".")          // look for config in the working directory

	// Read the .env file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			fmt.Println("No .env.local file found")
		} else {
			// Config file was found but another error was produced
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	return nil
}

func ReadJsonArrayConfig[T any](filePath string) ([]T, error) {
	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	// Unmarshal directly into slice
	var result []T
	if err := json.Unmarshal(content, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling config from %s: %w", filePath, err)
	}

	return result, nil
}

func ReadJsonConfig[T any](filePath string) (*T, error) {
	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	// Unmarshal directly into slice
	var result T
	if err := json.Unmarshal(content, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling config from %s: %w", filePath, err)
	}

	return &result, nil
}

func injectEnvConfig(cfg *Config) error {
	//Set config environment variables
	cfg.ConfigPath = viper.GetString("CONFIG_PATH")
	cfg.ChainEnv = viper.GetString("CHAIN_ENV")
	cfg.ConnnectionString = viper.GetString("DATABASE_URL")
	cfg.ScalarMnemonic = viper.GetString("SCALAR_MNEMONIC")
	cfg.EvmPrivateKey = viper.GetString("EVM_PRIVATE_KEY")
	cfg.BtcPrivateKey = viper.GetString("BTC_PRIVATE_KEY")
	return nil
}

func Load() error {
	// Load environment variables into viper
	if err := LoadEnv(); err != nil {
		panic("Failed to load environment variables: " + err.Error())
	}
	var cfg Config
	injectEnvConfig(&cfg)

	switch cfg.ChainEnv {
	case "local":
		fmt.Println("[getConfig] Using local configuration")
	case "devnet":
		fmt.Println("[getConfig] Using devnet configuration")
	case "testnet":
		fmt.Println("[getConfig] Using testnet configuration")
	default:
		return fmt.Errorf("[getConfig] Invalid CHAIN_ENV: %s", cfg.ChainEnv)
	}

	GlobalConfig = &cfg
	return nil
}
func GetScalarMnemonic() string {
	return GlobalConfig.ScalarMnemonic
}
