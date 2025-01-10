package scalar

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/clients/cosmos"
	"github.com/scalarorg/scalar-core/x/chains/types"
)

func subscribeTokenSentEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[*types.EventTokenSent]) error) error {
	if _, err := Subscribe(ctx, network, TokenSentEvent,
		func(events []IBCEvent[*types.EventTokenSent]) {
			err := callback(ctx, events)
			if err != nil {
				log.Error().Msgf("[ScalarClient] [TokenSendHandler] callback error: %v", err)
			}
		}); err != nil {
		log.Debug().Msgf("[ScalarClient] [TokenSendHandler] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [TokenSendHandler] success")
	}
	return nil
}
func subscribeMintCommand(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[*types.MintCommand]) error) error {
	if _, err := Subscribe(ctx, network, MintCommandEvent,
		func(events []IBCEvent[*types.MintCommand]) {
			err := callback(ctx, events)
			if err != nil {
				log.Error().Msgf("[ScalarClient] [ContractCallApprovedHandler] callback error: %v", err)
			}
		}); err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeContractCallApprovedEvent] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeContractCallApprovedEvent] success")
	}
	return nil
}
func subscribeContractCallWithTokenApprovedEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[*types.EventContractCallWithMintApproved]) error) error {
	if _, err := Subscribe(ctx, network, ContractCallWithMintApprovedEvent,
		func(events []IBCEvent[*types.EventContractCallWithMintApproved]) {
			err := callback(ctx, events)
			if err != nil {
				log.Error().Msgf("[ScalarClient] [ContractCallWithMintApprovedHandler] callback error: %v", err)
			}
		}); err != nil {
		log.Debug().Msgf("[ScalarClient] [ContractCallWithMintApprovedHandler] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [ContractCallWithMintApprovedHandler] success")
	}
	return nil
}
func subscribeContractCallApprovedEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[*types.ContractCallApproved]) error) error {
	if _, err := Subscribe(ctx, network, ContractCallApprovedEvent,
		func(events []IBCEvent[*types.ContractCallApproved]) {
			err := callback(ctx, events)
			if err != nil {
				log.Error().Msgf("[ScalarClient] [ContractCallApprovedHandler] callback error: %v", err)
			}
		}); err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeContractCallApprovedEvent] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeContractCallApprovedEvent] success")
	}
	return nil
}

func subscribeEVMCompletedEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[*types.ChainEventCompleted]) error) error {
	if _, err := Subscribe(ctx, network, EVMCompletedEvent,
		func(events []IBCEvent[*types.ChainEventCompleted]) {
			err := callback(ctx, events)
			if err != nil {
				log.Error().Msgf("[ScalarClient] [EVMCompletedHandler] callback error: %v", err)
			}
		}); err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeEVMCompletedEvent] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeEVMCompletedEvent] success")
	}
	return nil
}
func subscribeAllNewBlockEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[ScalarMessage]) error) error {
	//Subscribe to all events for debug purpose
	if _, err := Subscribe(ctx, network, AllNewBlockEvent,
		func(events []IBCEvent[ScalarMessage]) {
			err := callback(ctx, events)
			if err != nil {
				log.Error().Msgf("[ScalarClient] [AllNewBlockHandler] callback error: %v", err)
			}

		}); err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeAllNewBlockEvent] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeAllNewBlockEvent] success")
	}
	return nil
}

func subscribeAllTxEvent(ctx context.Context, network *cosmos.NetworkClient) error {
	//Subscribe to all events for debug purpose
	TxEvent := ListenerEvent[ScalarMessage]{
		Type: "Tx",
		//TopicId: "tm.event='Tx'",
		TopicId: "tm.event='*'",
		Parser: func(events map[string][]string) ([]IBCEvent[ScalarMessage], error) {
			log.Debug().Msgf("[ScalarClient] [AllTxHandler] events: %v", events)
			return nil, nil
		},
	}
	callback := func(events []IBCEvent[ScalarMessage]) {
		log.Debug().Msgf("[ScalarClient] [subscribeAllTxEvent] events: %v", events)
	}
	if _, err := Subscribe(ctx, network, TxEvent, callback); err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeAllTxEvent] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeAllTxEvent] success")
	}
	return nil
}

func subscribeCommandBatchSignedEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[*types.CommandBatchSigned]) error) error {
	if _, err := Subscribe(ctx, network, BatchCommandSignedEvent,
		func(events []IBCEvent[*types.CommandBatchSigned]) {
			err := callback(ctx, events)
			if err != nil {
				log.Error().Msgf("[ScalarClient] [subscribeCommandBatchSignedEvent] callback error: %v", err)
			}

		}); err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeCommandBatchSignedEvent] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeCommandBatchSignedEvent] success")
	}
	return nil
}
