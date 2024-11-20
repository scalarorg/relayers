package evm_test

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/clients/evm"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
)

var (
	globalConfig *config.Config        = &config.Config{}
	evmConfig    *evm.EvmNetworkConfig = &evm.EvmNetworkConfig{
		ChainID:   11155111,
		ID:        "ethereum-sepolia",
		Name:      "Ethereum sepolia",
		RPCUrl:    "wss://eth-sepolia.g.alchemy.com/v2/nNbspp-yjKP9GtAcdKi8xcLnBTptR2Zx",
		Gateway:   "0x2bb588d7bb6faAA93f656C3C78fFc1bEAfd1813D",
		Finality:  1,
		LastBlock: 7061000,
	}
	evmClient *evm.EvmClient
)

//	func TestMain(m *testing.M) {
//		var err error
//		log.Info().Msgf("Creating evm client with config: %v", evmConfig)
//		evmClient, err = evm.NewEvmClient(globalConfig, evmConfig, nil, nil)
//		if err != nil {
//			log.Error().Msgf("failed to create evm client: %v", err)
//		}
//		os.Exit(m.Run())
//	}
func TestEvmClientListenContractCallEvent(t *testing.T) {
	watchOpts := bind.WatchOpts{Start: &evmConfig.LastBlock, Context: context.Background()}
	sink := make(chan *contracts.IAxelarGatewayContractCall)

	subContractCall, err := evmClient.Gateway.WatchContractCall(&watchOpts, sink, nil, nil)
	if err != nil {
		log.Error().Msgf("failed to watch ContractCallEvent: %v", err)
	}
	if subContractCall != nil {
		log.Info().Msg("Subscribed to ContractCallEvent successfully. Waiting for events...")
		go func() {
			for event := range sink {
				log.Info().Msgf("Contract call: %v", event)
			}
		}()
		go func() {
			errChan := subContractCall.Err()
			if err := <-errChan; err != nil {
				log.Error().Msgf("Received error: %v", err)
			}
		}()
		defer subContractCall.Unsubscribe()
	}
	select {}
}
func TestEvmClientListenContractCallApprovedEvent(t *testing.T) {

}
