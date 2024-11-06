package evm_test

import (
	"testing"

	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/services/evm"
	"github.com/scalarorg/relayers/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestContractCallApprovedListenerWithDB(t *testing.T) {
	eventChan := make(chan *types.EventEnvelope, 100)

	// Setup test database
	dbAdapter, cleanup, err := db.SetupTestDB(eventChan)
	require.NoError(t, err)
	defer cleanup() // This ensures the container is cleaned up after the test

	evmListener, err := evm.NewEvmAdapter([]config.EvmNetworkConfig{
		{
			Name:       "sepolia",
			RPCUrl:     "https://eth-sepolia.g.alchemy.com/v2/nNbspp-yjKP9GtAcdKi8xcLnBTptR2Zx",
			Gateway:    "0x2bb588d7bb6faAA93f656C3C78fFc1bEAfd1813D",
			PrivateKey: "81271046a6de40cea0798935b391ddcf1e57646db7273228fd0d7e147154aaaa",
			ChainID:    "11155111",
		},
	}, eventChan)
	require.NoError(t, err)

	// Start listening for events in a separate goroutine
	go func() {
		for event := range eventChan {
			t.Logf("Received event: %+v", event)
			if event.Component == "DbAdapter" {
				dbAdapter.EventChan <- event
			} else if event.Component == "EvmAdapter" {
				evmListener.EventChan <- event
			}
		}
	}()

	// Start listening for database events
	go dbAdapter.ListenEvents()

	go evmListener.ListenEvents()

	evmListener.PollForContractCallApproved()
}
