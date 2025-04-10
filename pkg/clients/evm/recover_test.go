package evm_test

import (
	"context"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/clients/evm"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	chainExported "github.com/scalarorg/scalar-core/x/chains/exported"
	covExported "github.com/scalarorg/scalar-core/x/covenant/exported"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

var (
	mockCustodianGroupUid = sha3.Sum256([]byte("scalarv32"))
	mockCustodianGroup    = &covExported.CustodianGroup{
		UID: chainExported.Hash(mockCustodianGroupUid[:]),
	}
)

func TestSepoliaRecoverEvents(t *testing.T) {
	sepoliaClient, err := evm.NewEvmClient(&globalConfig, sepoliaConfig, nil, nil, nil)
	if err != nil {
		log.Error().Msgf("failed to create evm client: %v", err)
	}
	wg := sync.WaitGroup{}
	//Log missing logs
	wg.Add(1)
	go func() {
		defer wg.Done()
		scalarGatewayAbi, _ := contracts.IScalarGatewayMetaData.GetAbi()
		events := map[string]abi.Event{}
		for _, event := range scalarGatewayAbi.Events {
			events[event.ID.String()] = event
		}
		for !sepoliaClient.MissingLogs.IsRecovered() {
			logs := sepoliaClient.MissingLogs.GetLogs(10)
			for _, txLog := range logs {
				topic := txLog.Topics[0].String()
				event, ok := events[topic]
				if !ok {
					log.Error().Str("topic", topic).Any("txLog", txLog).Msg("[EvmClient] [ProcessMissingLogs] event not found")
					continue
				}
				log.Debug().
					Str("chainId", sepoliaClient.EvmConfig.GetId()).
					Str("eventName", event.Name).
					Str("txHash", txLog.TxHash.String()).
					Msg("[EvmClient] [ProcessMissingLogs] processing missing event")
			}
		}
		log.Info().Str("Chain", sepoliaClient.EvmConfig.ID).Msg("[EvmClient] [ProcessMissingLogs] finished processing all missing evm events")

	}()

	err = sepoliaClient.RecoverAllEvents(context.Background(), []*covExported.CustodianGroup{mockCustodianGroup})
	require.NoError(t, err)
	wg.Wait()
}

func TestSepoliaProcessMissingLogs(t *testing.T) {
	sepoliaClient, err := evm.NewEvmClient(&globalConfig, sepoliaConfig, nil, nil, nil)
	if err != nil {
		log.Error().Msgf("failed to create evm client: %v", err)
	}
	wg := sync.WaitGroup{}
	//Log missing logs
	wg.Add(1)
	go func() {
		defer wg.Done()
		sepoliaClient.ProcessMissingLogs()
	}()
	//err = sepoliaClient.RecoverEvents(context.Background(), []string{events.EVENT_EVM_SWITCHED_PHASE})
	err = sepoliaClient.RecoverAllEvents(context.Background(), []*covExported.CustodianGroup{mockCustodianGroup})
	require.NoError(t, err)
	wg.Wait()
}
