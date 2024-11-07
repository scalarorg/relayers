package config

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

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

type EvmNetworkConfig struct {
	ChainID    string `mapstructure:"chain_id"`
	ID         string `mapstructure:"id"`
	Name       string `mapstructure:"name"`
	RPCUrl     string `mapstructure:"rpc_url"`
	Gateway    string `mapstructure:"gateway"`
	Finality   int    `mapstructure:"finality"`
	LastBlock  string `mapstructure:"last_block"`
	PrivateKey string `mapstructure:"private_key"`
	MaxRetry   int
	RetryDelay time.Duration
	TxTimeout  time.Duration
}

type CosmosNetworkConfig struct {
	ChainID  string  `mapstructure:"chain_id"`
	RPCUrl   string  `mapstructure:"rpc_url"`
	LCDUrl   string  `mapstructure:"lcd_url"`
	WS       string  `mapstructure:"ws"`
	Denom    string  `mapstructure:"denom"`
	Mnemonic *string `mapstructure:"mnemonic"`
	GasPrice string  `mapstructure:"gas_price"`
}

type BtcNetworkConfig struct {
	Network    string  `mapstructure:"network"`
	ID         string  `mapstructure:"id"`
	ChainID    string  `mapstructure:"chain_id"`
	Name       string  `mapstructure:"name"`
	Type       string  `mapstructure:"type"`
	Host       string  `mapstructure:"host"`
	Port       int     `mapstructure:"port"`
	User       string  `mapstructure:"user"`
	Password   string  `mapstructure:"password"`
	SSL        *bool   `mapstructure:"ssl,omitempty"`
	PrivateKey *string `mapstructure:"private_key,omitempty"`
	Address    *string `mapstructure:"address,omitempty"`
	Gateway    *string `mapstructure:"gateway,omitempty"`
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
	RabbitMQ      RabbitMQConfig        `mapstructure:"rabbitmq"`
	Axelar        CosmosNetworkConfig   `mapstructure:"axelar"`
	EvmNetworks   []EvmNetworkConfig    `mapstructure:"evm_networks"`
	CosmosNetwork []CosmosNetworkConfig `mapstructure:"cosmos_network"`
	BtcNetworks   []BtcNetworkConfig    `mapstructure:"btc_networks"`
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

func Load() error {
	var cfg Config

	configPath := viper.GetString("CONFIG_PATH")
	chainEnv := viper.GetString("CHAIN_ENV")
	dir := fmt.Sprintf("%s/%s", configPath, chainEnv)

	switch chainEnv {
	case "local":
		fmt.Println("[getConfig] Using local configuration")
	case "devnet":
		fmt.Println("[getConfig] Using devnet configuration")
	case "testnet":
		fmt.Println("[getConfig] Using testnet configuration")
	default:
		return fmt.Errorf("[getConfig] Invalid CHAIN_ENV: %s", chainEnv)
	}

	// Read Axelar config from JSON file
	axelarContent, err := os.ReadFile(fmt.Sprintf("%s/axelar.json", dir))
	if err != nil {
		return fmt.Errorf("error reading Axelar config file: %w", err)
	}

	var axelarConfig struct {
		Axelar CosmosNetworkConfig `json:"axelar"`
	}
	if err := json.Unmarshal(axelarContent, &axelarConfig); err != nil {
		return fmt.Errorf("error unmarshaling Axelar config: %w", err)
	}
	cfg.Axelar = axelarConfig.Axelar

	axelarMnemonic := viper.GetString("AXELAR_MNEMONIC")
	cfg.Axelar.Mnemonic = &axelarMnemonic

	// Read BTC broadcast config
	btcBroadcastConfig, err := ReadJsonArrayConfig[BtcNetworkConfig](fmt.Sprintf("%s/btc.json", dir))
	if err != nil {
		return fmt.Errorf("error reading BTC broadcast config: %w", err)
	}

	// Read BTC signer config
	btcSignerConfig, err := ReadJsonArrayConfig[BtcNetworkConfig](fmt.Sprintf("%s/btc-signer.json", dir))
	if err != nil {
		return fmt.Errorf("error reading BTC signer config: %w", err)
	}

	// Combine BTC configs
	cfg.BtcNetworks = append(btcBroadcastConfig, btcSignerConfig...)

	for i := range cfg.BtcNetworks {
		privateKey := viper.GetString("BTC_PRIVATE_KEY")
		cfg.BtcNetworks[i].PrivateKey = &privateKey
	}

	// Read Cosmos network config from JSON file
	cosmosNetworks, err := ReadJsonArrayConfig[CosmosNetworkConfig](fmt.Sprintf("%s/cosmos.json", dir))
	if err != nil {
		return fmt.Errorf("error reading Cosmos network config: %w", err)
	}
	cfg.CosmosNetwork = cosmosNetworks

	cosmosMnemonic := viper.GetString("AXELAR_MNEMONIC")
	for i := range cfg.CosmosNetwork {
		cfg.CosmosNetwork[i].Mnemonic = &cosmosMnemonic
	}

	// Read EVM Network configs
	evmNetworks, err := ReadJsonArrayConfig[EvmNetworkConfig](fmt.Sprintf("%s/evm.json", dir))
	if err != nil {
		return fmt.Errorf("error reading EVM network config: %w", err)
	}

	// Get EVM private keys and assign them to the corresponding networks
	for i, evmNetwork := range evmNetworks {
		privateKey, err := GetEvmPrivateKey(evmNetwork.ID)
		if err != nil {
			return fmt.Errorf("failed to get EVM private key for network %s: %w", evmNetwork.ID, err)
		}
		evmNetworks[i].PrivateKey = privateKey
	}

	cfg.EvmNetworks = evmNetworks

	// // Read RabbitMQ config from JSON file
	// viper.SetConfigFile(fmt.Sprintf("%s/rabbitmq.json", dir))
	// viper.SetConfigType("json")

	// if err := viper.ReadInConfig(); err != nil {
	// 	return fmt.Errorf("error reading RabbitMQ config: %w", err)
	// }

	// if err := viper.UnmarshalKey("rabbitmq", &cfg.RabbitMQ); err != nil {
	// 	return fmt.Errorf("error unmarshaling RabbitMQ config: %w", err)
	// }

	GlobalConfig = &cfg
	return nil
}

func GetEvmPrivateKey(networkID string) (string, error) {
	configChainsPath := viper.GetString("CONFIG_CHAINS_RUNTIME_PATH")
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
		privateKey = hex.EncodeToString(privateKeyBytes)
	}

	if privateKey == "" {
		return "", fmt.Errorf("no private key found for network %s", networkConfig.ID)
	}

	return privateKey, nil
}
