package btc_test

import (
	"os"
	"testing"

	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/clients/btc"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/stretchr/testify/assert"
)

var (
	TAPROOT_ADDRESS string = "tb1p07q440mdl4uyywns325dk8pvjphwety3psp4zvkngtjf3z3hhr2sfar3hv"
	btcClient       *btc.BtcClient
	globalConfig    *config.Config
	btcConfig       *btc.BtcNetworkConfig
	dbAdapter       *db.DatabaseAdapter
	eventBus        *events.EventBus
)

func TestMain(m *testing.M) {
	globalConfig = &config.Config{}
	btcConfig = &btc.BtcNetworkConfig{
		Name:       "test",
		Host:       "testnet4.btc.scalar.org",
		Port:       80,
		User:       "user",
		Password:   "password",
		SSL:        nil,
		MempoolUrl: "https://mempool.space/testnet4/api",
		Address:    &TAPROOT_ADDRESS,
	}
	btcClient, _ = btc.NewBtcClientFromConfig(
		globalConfig,
		btcConfig,
		dbAdapter,
		eventBus,
	)
	os.Exit(m.Run())
}

func TestGetAddressTxsUtxo(t *testing.T) {
	utxos, err := btcClient.GetAddressTxsUtxo(TAPROOT_ADDRESS)
	assert.NoError(t, err)
	assert.NotEmpty(t, utxos)
}
