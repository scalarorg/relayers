package electrs

type Config struct {
	//Electrum server host
	Host string
	//Electrum server port
	Port int
	//Electrum server user
	User string
	//Electrum server password
	Password string
	//Source chain - This must match with bridge config in the xchains core config. For example bitcoin-testnet4
	SourceChain string
	//Destination chain
	DestinationChain string
	//Las Vault Tx's hash received from electrum server.
	//If this parameter is empty, server will start from the first vault tx from db.
	BatchSize   int
	LastVaultTx string
}
