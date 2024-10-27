package config

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
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

type AxelarConfig struct {
	ChainID  string `mapstructure:"chainId"`
	Denom    string `mapstructure:"denom"`
	GasPrice string `mapstructure:"gasPrice"`
	RPCUrl   string `mapstructure:"rpcUrl"`
	LCDUrl   string `mapstructure:"lcdUrl"`
	WS       string `mapstructure:"ws"`
}

type DatabaseConfig struct {
	URL string `mapstructure:"url"`
}

type EvmNetworkConfig struct {
	ChainID    string `mapstructure:"chain_id"`
	ID         string `mapstructure:"id"`
	Name       string `mapstructure:"name"`
	RPCUrl     string `mapstructure:"rpc_url"`
	Gateway    string `mapstructure:"gateway"`
	Finality   int    `mapstructure:"finality"`
	PrivateKey string `mapstructure:"private_key"`
}

type CosmosNetworkConfig struct {
	ChainID  string `mapstructure:"chain_id"`
	RPCUrl   string `mapstructure:"rpc_url"`
	LCDUrl   string `mapstructure:"lcd_url"`
	WS       string `mapstructure:"ws"`
	Denom    string `mapstructure:"denom"`
	Mnemonic string `mapstructure:"mnemonic"`
	GasPrice string `mapstructure:"gas_price"`
}

type BtcNetworkConfig struct {
	Network    string  `mapstructure:"network"`
	ID         string  `mapstructure:"id"`
	ChainID    string  `mapstructure:"chain_id"`
	Type       string  `mapstructure:"type"` // signer or broadcast
	Host       string  `mapstructure:"host"`
	Port       int     `mapstructure:"port"`
	User       string  `mapstructure:"user"`
	Password   string  `mapstructure:"password"`
	SSL        *bool   `mapstructure:"ssl,omitempty"`
	PrivateKey *string `mapstructure:"private_key,omitempty"`
	Address    *string `mapstructure:"address,omitempty"`
}

type RuntimeEvmNetworkConfig struct {
	ChainID       string  `mapstructure:"chainId"`
	ID            string  `mapstructure:"id"`
	Name          string  `mapstructure:"name"`
	MnemonicEvm   *string `mapstructure:"mnemonic_evm,omitempty"`
	Mnemonic      *string `mapstructure:"mnemonic,omitempty"`
	WalletIndex   *string `mapstructure:"walletIndex,omitempty"`
	RPCUrl        string  `mapstructure:"rpcUrl"`
	KeyPath       string  `mapstructure:"keyPath"`
	PrivateKey    *string `mapstructure:"privateKey,omitempty"`
	Gateway       *string `mapstructure:"gateway,omitempty"`
	GasService    *string `mapstructure:"gasService,omitempty"`
	AuthWeighted  *string `mapstructure:"authWeighted,omitempty"`
	TokenDeployer *string `mapstructure:"tokenDeployer,omitempty"`
	SBtc          *string `mapstructure:"sBtc,omitempty"`
}

type Config struct {
	RabbitMQ      RabbitMQConfig      `mapstructure:"rabbitmq"`
	Axelar        AxelarConfig        `mapstructure:"axelar"`
	Database      DatabaseConfig      `mapstructure:"database"`
	EvmNetworks   []EvmNetworkConfig  `mapstructure:"evm_networks"`
	CosmosNetwork CosmosNetworkConfig `mapstructure:"cosmos_network"`
	BtcNetwork    BtcNetworkConfig    `mapstructure:"btc_network"`
}

var GlobalConfig *Config

func LoadEnv() error {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		return err
	}

	return nil
}

func Load() error {
	var cfg Config

	// --- Read RabbitMQ config from JSON file
	viper.SetConfigFile("data/example-env/rabbitmq.json")
	viper.SetConfigType("json")

	if err := viper.MergeInConfig(); err != nil {
		return err
	}

	// Unmarshal RabbitMQ config
	if err := viper.UnmarshalKey("rabbitmq", &cfg.RabbitMQ); err != nil {
		return err
	}

	// --- Read Axelar config from JSON file
	viper.SetConfigFile("data/example-env/axelar.json")
	viper.SetConfigType("json")

	if err := viper.MergeInConfig(); err != nil {
		return err
	}

	// Unmarshal Axelar config
	if err := viper.UnmarshalKey("axelar", &cfg.Axelar); err != nil {
		return err
	}

	// Load EVM Network configs
	viper.SetConfigFile("data/example-env/evm.json")
	viper.SetConfigType("json")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading EVM config file: %w", err)
	}

	var evmNetworks []EvmNetworkConfig
	if err := viper.Unmarshal(&evmNetworks); err != nil {
		return fmt.Errorf("error unmarshaling EVM networks config: %w", err)
	}

	// Get EVM private keys and assign them to the corresponding networks
	for i, evmNetwork := range evmNetworks {
		privateKey, err := cfg.GetEvmPrivateKey(evmNetwork.ID)
		if err != nil {
			return fmt.Errorf("failed to get EVM private key for network %s: %w", evmNetwork.ID, err)
		}
		evmNetworks[i].PrivateKey = privateKey
	}

	cfg.EvmNetworks = evmNetworks

	// --- Read Cosmos network config from JSON file
	viper.SetConfigFile("data/example-env/cosmos.json")
	viper.SetConfigType("json")

	if err := viper.MergeInConfig(); err != nil {
		return err
	}

	// Unmarshal Cosmos network config
	if err := viper.UnmarshalKey("cosmos_network", &cfg.CosmosNetwork); err != nil {
		return err
	}

	// Load database URL from environment variable (already set by LoadEnv)
	cfg.Database.URL = viper.GetString("DATABASE_URL")

	GlobalConfig = &cfg
	return nil
}

func (c *Config) GetEvmPrivateKey(networkID string) (string, error) {
	configChainsPath := viper.GetString("CONFIG_CHAINS")
	configFile := fmt.Sprintf("%s/%s/config.json", configChainsPath, networkID)

	// Check if config.json exists for the network
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return "", fmt.Errorf("no config file found for network ID %s", networkID)
	}

	// Create a new Viper instance
	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("json")

	if err := v.ReadInConfig(); err != nil {
		return "", fmt.Errorf("error reading config file %s: %w", configFile, err)
	}

	var networkConfig RuntimeEvmNetworkConfig
	if err := v.Unmarshal(&networkConfig); err != nil {
		return "", fmt.Errorf("error unmarshaling config from %s: %w", configFile, err)
	}

	var privateKey string

	if networkConfig.PrivateKey != nil && *networkConfig.PrivateKey != "" {
		privateKey = *networkConfig.PrivateKey
	} else if networkConfig.Mnemonic != nil && networkConfig.WalletIndex != nil {
		wallet, err := hdwallet.NewFromMnemonic(*networkConfig.Mnemonic)
		if err != nil {
			return "", fmt.Errorf("failed to create wallet from mnemonic: %w", err)
		}

		path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%s", *networkConfig.WalletIndex))
		account, err := wallet.Derive(path, true)
		if err != nil {
			return "", fmt.Errorf("failed to derive account: %w", err)
		}

		privateKeyECDSA, err := wallet.PrivateKey(account)
		if err != nil {
			return "", fmt.Errorf("failed to get private key: %w", err)
		}

		privateKeyBytes := crypto.FromECDSA(privateKeyECDSA)
		privateKey = "0x" + hex.EncodeToString(privateKeyBytes)
	}

	if privateKey == "" {
		return "", fmt.Errorf("no private key found for network %s", networkConfig.ID)
	}

	// You might want to use a proper logging library instead of fmt.Printf
	fmt.Printf("[GetEvmPrivateKey] network: %s, privateKey: %s\n", networkConfig.ID, privateKey)

	return privateKey, nil
}
