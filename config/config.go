package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type EventBusConfig struct {
}

type IChainConfig interface {
	GetId() string      //String identifier for the chain for example ethereum-sepolia
	GetChainId() uint64 //Integer identifier for the chain for example 11155111
	GetFamily() string  //Family of the chain for example evm
	GetName() string    //Name of the chain for example Ethereum Sepolia
}
type ChainFamily map[uint64]IChainConfig
type Config struct {
	ConfigPath        string                 `mapstructure:"config_path"`
	ConnnectionString string                 `mapstructure:"database_url"` // Postgres db connection string
	IndexerUrl        string                 `mapstructure:"indexer_url"`
	ScalarMnemonic    string                 `mapstructure:"scalar_mnemonic"`
	EvmKey            string                 `mapstructure:"evm_key"`
	Nonce             string                 `mapstructure:"nonce"`
	BtcPrivateKey     string                 `mapstructure:"btc_private_key"`
	ChainConfigs      map[string]ChainFamily `mapstructure:"chain_configs"` //Store all valid chain configs
	//ActiveChains      map[string]bool        `mapstructure:"active_chains"` //Store all active chains in the scalar network
}

const (
	APP_NAME = "scalar"
)

var GlobalConfig Config

func LoadEnv(environment string) error {
	// Tell Viper to read from environment
	viper.AutomaticEnv()
	// Add support for .env files
	if environment == "" {
		viper.SetConfigFile(".env") // Set config file
	} else {
		viper.SetConfigName(environment)
		viper.SetConfigType("env")
	}
	viper.AddConfigPath(".") // look for config in the working directory
	log.Info().Msgf("[LoadEnv] ReadInConfig for environment: %s", environment)
	// Read the .env file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			fmt.Printf("No %s.env file found", environment)
		} else {
			// Config file was found but another error was produced
			return fmt.Errorf("error reading config file: %w", err)
		}
	}
	viper.Unmarshal(&GlobalConfig)
	if GlobalConfig.ScalarMnemonic == "" {
		GlobalConfig.ScalarMnemonic = os.Getenv("SCALAR_MNEMONIC")
	}
	// Initialize an empty chain configs map
	GlobalConfig.ChainConfigs = make(map[string]ChainFamily)
	log.Info().Msgf("Loaded config: %+v", GlobalConfig)
	//injectEnvConfig(&GlobalConfig)
	return nil
}

func ReadJsonArrayConfig[T any](filePath string) ([]T, error) {
	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	// Unmarshal directly into slice
	result, err := ParseJsonArrayConfig[T](content)
	if err != nil {
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
	result, err := ParseJsonConfig[T](content)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling config from %s: %w", filePath, err)
	}

	return result, nil
}

func ParseJsonArrayConfig[T any](content []byte) ([]T, error) {
	var result []T
	if err := json.Unmarshal(content, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func ParseJsonConfig[T any](content []byte) (*T, error) {
	var result T
	if err := json.Unmarshal(content, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// func injectEnvConfig(cfg *Config) error {
// 	//Set config environment variables
// 	cfg.ConfigPath = viper.GetString("CONFIG_PATH")
// 	cfg.ConnnectionString = viper.GetString("DATABASE_URL")
// 	cfg.ScalarMnemonic = viper.GetString("SCALAR_MNEMONIC")
// 	cfg.EvmPrivateKey = viper.GetString("EVM_PRIVATE_KEY")
// 	cfg.BtcPrivateKey = viper.GetString("BTC_PRIVATE_KEY")
// 	return nil
// }

func (c *Config) AddChainConfig(chainConfig IChainConfig) {
	family := chainConfig.GetFamily()
	if _, ok := c.ChainConfigs[family]; !ok {
		c.ChainConfigs[family] = make(ChainFamily)
	}
	c.ChainConfigs[family][chainConfig.GetChainId()] = chainConfig
}
func (c *Config) GetStringIdByChainId(chainFamily string, chainId uint64) (string, error) {
	// log.Debug().Msgf("Getting string id for chainId: %d", chainId)
	family, ok := c.ChainConfigs[chainFamily]
	if !ok {
		return "", fmt.Errorf("chain not found for chainId: %d", chainId)
	}
	chainConfig, ok := family[chainId]
	if !ok {
		return "", fmt.Errorf("chain not found for chainId: %d", chainId)
	}
	return chainConfig.GetId(), nil
}

func (c *Config) GetChainConfigById(chainFamily string, chainId uint64) (IChainConfig, error) {
	family, ok := c.ChainConfigs[chainFamily]
	if !ok {
		return nil, fmt.Errorf("chain not found for chainId: %d", chainId)
	}
	chainConfig, ok := family[chainId]
	if !ok {
		return nil, fmt.Errorf("chain not found for chainId: %d", chainId)
	}
	return chainConfig, nil
}
