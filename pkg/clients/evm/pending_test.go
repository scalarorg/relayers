package evm_test

import (
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/clients/evm/pending"
	"github.com/stretchr/testify/require"
)

func TestPollTxForEvents(t *testing.T) {
	pendingTx := pending.PendingTx{
		TxHash:    "0xd372c87a830857662a6fb37ee1bea1b8465cc5469e90d0ecee4020ff284bb149",
		Timestamp: time.Now().Add(time.Minute * -1),
	}
	evmClient.AddPendingTx(pendingTx.TxHash, pendingTx.Timestamp)
	events, err := evmClient.PollTxForEvents(pendingTx)
	require.NoError(t, err)
	require.NotNil(t, events)
	require.NotNil(t, events.ContractCallApproved)
	require.NotNil(t, events.Executed)
	log.Debug().Msgf("[TestPollTxForEvents] ContractCallApproved: %+v", events.ContractCallApproved)
	// log.Debug().Msgf("[TestPollTxForEvents] ContractCall: %+v", events.ContractCall)
	// log.Debug().Msgf("[TestPollTxForEvents] Executed: %+v", events.Executed)
}
