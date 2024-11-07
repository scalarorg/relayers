package evm_test

import (
	"testing"
	"time"

	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/adapters/evm"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestContractCallEventListenerWithDB(t *testing.T) {
	var (
		channelBufferSize = 100
		chainName         = "sepolia"
		rpcUrl            = "https://eth-sepolia.g.alchemy.com/v2/nNbspp-yjKP9GtAcdKi8xcLnBTptR2Zx"
		gateway           = "0x2bb588d7bb6faAA93f656C3C78fFc1bEAfd1813D"
		privateKey        = "81271046a6de40cea0798935b391ddcf1e57646db7273228fd0d7e147154aaaa"
		chainID           = "11155111"
	)

	busEventChan := make(chan *types.EventEnvelope, channelBufferSize)

	// Setup test database
	dbAdapter, cleanup, err := db.SetupTestDB(busEventChan, channelBufferSize)
	require.NoError(t, err)
	defer cleanup() // This ensures the container is cleaned up after the test

	evmListener, err := evm.NewEvmAdapter([]config.EvmNetworkConfig{
		{
			Name:       chainName,
			RPCUrl:     rpcUrl,
			Gateway:    gateway,
			PrivateKey: privateKey,
			ChainID:    chainID,
			TxTimeout:  time.Second * 10,
		},
	}, busEventChan, channelBufferSize)
	require.NoError(t, err)

	// Start listening for events in a separate goroutine
	go func() {
		for event := range busEventChan {
			t.Logf("Received event: %+v", event)
			if event.Component == "DbAdapter" {
				dbAdapter.BusEventReceiverChan <- event
			} else if event.Component == "EvmAdapter" {
				evmListener.BusEventReceiverChan <- event
			}
		}
	}()

	// Start listening for database events
	go dbAdapter.ListenEventsFromBusChannel()

	go evmListener.ListenEventsFromBusChannel()

	evmListener.PollForContractCall()
}
