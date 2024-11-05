package config

import (
	"encoding/hex"
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
	RabbitMQ      RabbitMQConfig      `mapstructure:"rabbitmq"`
	Axelar        CosmosNetworkConfig `mapstructure:"axelar"`
	EvmNetworks   []EvmNetworkConfig  `mapstructure:"evm_networks"`
	CosmosNetwork CosmosNetworkConfig `mapstructure:"cosmos_network"`
	BtcNetworks   []BtcNetworkConfig  `mapstructure:"btc_networks"`
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

	chainEnv := viper.GetString("CHAIN_ENV")
	dir := fmt.Sprintf("data/%s", chainEnv)

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
	viper.SetConfigFile(fmt.Sprintf("%s/axelar.json", dir))
	viper.SetConfigType("json")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading Axelar config: %w", err)
	}

	if err := viper.UnmarshalKey("axelar", &cfg.Axelar); err != nil {
		return fmt.Errorf("error unmarshaling Axelar config: %w", err)
	}

	axelarMnemonic := viper.GetString("AXELAR_MNEMONIC")
	cfg.Axelar.Mnemonic = &axelarMnemonic

	// Read BTC broadcast config
	viper.SetConfigFile(fmt.Sprintf("%s/btc.json", dir))
	viper.SetConfigType("json")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading BTC broadcast config: %w", err)
	}

	var btcBroadcastConfig []BtcNetworkConfig
	if err := viper.Unmarshal(&btcBroadcastConfig); err != nil {
		return fmt.Errorf("error unmarshaling BTC broadcast config: %w", err)
	}

	// Read BTC signer config
	viper.SetConfigFile(fmt.Sprintf("%s/btc-signer.json", dir))
	viper.SetConfigType("json")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading BTC signer config: %w", err)
	}

	var btcSignerConfig []BtcNetworkConfig
	if err := viper.Unmarshal(&btcSignerConfig); err != nil {
		return fmt.Errorf("error unmarshaling BTC signer config: %w", err)
	}

	// Combine BTC configs
	cfg.BtcNetworks = append(btcBroadcastConfig, btcSignerConfig...)

	for i := range cfg.BtcNetworks {
		privateKey := viper.GetString("BTC_PRIVATE_KEY")
		cfg.BtcNetworks[i].PrivateKey = &privateKey
	}

	// Read Cosmos network config from JSON file
	viper.SetConfigFile(fmt.Sprintf("%s/cosmos.json", dir))
	viper.SetConfigType("json")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading Cosmos config: %w", err)
	}

	if err := viper.UnmarshalKey("cosmos_network", &cfg.CosmosNetwork); err != nil {
		return fmt.Errorf("error unmarshaling Cosmos network config: %w", err)
	}

	cosmosMnemonic := viper.GetString("AXELAR_MNEMONIC")
	cfg.CosmosNetwork.Mnemonic = &cosmosMnemonic

	// Read EVM Network configs
	viper.SetConfigFile(fmt.Sprintf("%s/evm.json", dir))
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
		privateKey, err := GetEvmPrivateKey(evmNetwork.ID)
		if err != nil {
			return fmt.Errorf("failed to get EVM private key for network %s: %w", evmNetwork.ID, err)
		}
		evmNetworks[i].PrivateKey = privateKey
	}

	cfg.EvmNetworks = evmNetworks

	// Read RabbitMQ config from JSON file
	viper.SetConfigFile(fmt.Sprintf("%s/rabbitmq.json", dir))
	viper.SetConfigType("json")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading RabbitMQ config: %w", err)
	}

	if err := viper.UnmarshalKey("rabbitmq", &cfg.RabbitMQ); err != nil {
		return fmt.Errorf("error unmarshaling RabbitMQ config: %w", err)
	}

	GlobalConfig = &cfg
	return nil
}

func GetEvmPrivateKey(networkID string) (string, error) {
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
