package scalar

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/clients/cosmos"
	"github.com/scalarorg/scalar-core/x/chains/types"
)

var EventTimeout = 5 * time.Minute

func subscribeTokenSentEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[*types.EventTokenSent]) error) error {
	internalCallback := func(events []IBCEvent[*types.EventTokenSent]) {
		err := callback(ctx, events)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [TokenSendHandler] callback error: %v", err)
		}
	}
	_, err := SubscribeWithTimeout(ctx, EventTimeout, network, TokenSentEvent, internalCallback)
	// _, err := Subscribe(ctx, network, TokenSentEvent,internalCallback)
	if err != nil {
		log.Debug().Msgf("[ScalarClient] [TokenSendHandler] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [TokenSendHandler] success")
	}
	return nil
}
func subscribeMintCommand(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[*types.MintCommand]) error) error {
	internalCallback := func(events []IBCEvent[*types.MintCommand]) {
		err := callback(ctx, events)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [MintCommandHandler] callback error: %v", err)
		}
	}
	_, err := SubscribeWithTimeout(ctx, EventTimeout, network, MintCommandEvent, internalCallback)
	// _, err := Subscribe(ctx, network, MintCommandEvent, internalCallback)
	if err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeMintCommand] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeMintCommand] success")
	}
	return nil
}
func subscribeContractCallWithTokenApprovedEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[*types.EventContractCallWithMintApproved]) error) error {
	internalCallback := func(events []IBCEvent[*types.EventContractCallWithMintApproved]) {
		err := callback(ctx, events)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [ContractCallWithMintApprovedHandler] callback error: %v", err)
		}
	}
	_, err := SubscribeWithTimeout(ctx, EventTimeout, network, ContractCallWithMintApprovedEvent, internalCallback)
	// _, err := Subscribe(ctx, network, ContractCallWithMintApprovedEvent, internalCallback)
	if err != nil {
		log.Debug().Msgf("[ScalarClient] [ContractCallWithMintApprovedHandler] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [ContractCallWithMintApprovedHandler] success")
	}
	return nil
}
func subscribeContractCallApprovedEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[*types.ContractCallApproved]) error) error {
	internalCallback := func(events []IBCEvent[*types.ContractCallApproved]) {
		err := callback(ctx, events)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [ContractCallApprovedHandler] callback error: %v", err)
		}
	}
	_, err := SubscribeWithTimeout(ctx, EventTimeout, network, ContractCallApprovedEvent, internalCallback)
	// _, err := Subscribe(ctx, network, ContractCallApprovedEvent, internalCallback)
	if err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeContractCallApprovedEvent] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeContractCallApprovedEvent] success")
	}
	return nil
}

func subscribeEVMCompletedEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[*types.ChainEventCompleted]) error) error {
	internalCallback := func(events []IBCEvent[*types.ChainEventCompleted]) {
		err := callback(ctx, events)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [EVMCompletedHandler] callback error: %v", err)
		}
	}
	_, err := SubscribeWithTimeout(ctx, 5*time.Minute, network, EVMCompletedEvent, internalCallback)
	// _, err := Subscribe(ctx, network, EVMCompletedEvent, internalCallback)
	if err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeEVMCompletedEvent] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeEVMCompletedEvent] success")
	}
	return nil
}
func subscribeAllNewBlockEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[ScalarMessage]) error) error {
	internalCallback := func(events []IBCEvent[ScalarMessage]) {
		err := callback(ctx, events)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [AllNewBlockHandler] callback error: %v", err)
		}
	}
	_, err := SubscribeWithTimeout(ctx, EventTimeout, network, AllNewBlockEvent, internalCallback)
	// _, err := Subscribe(ctx, network, AllNewBlockEvent, internalCallback)
	if err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeAllNewBlockEvent] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeAllNewBlockEvent] success")
	}
	return nil
}

func subscribeAllTxEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[ScalarMessage]) error) error {
	internalCallback := func(events []IBCEvent[ScalarMessage]) {
		err := callback(ctx, events)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [AllTxHandler] callback error: %v", err)
		}
	}
	TxEvent := ListenerEvent[ScalarMessage]{
		Type: "Tx",
		//TopicId: "tm.event='Tx'",
		TopicId: "tm.event='*'",
		Parser: func(events map[string][]string) ([]IBCEvent[ScalarMessage], error) {
			log.Debug().Msgf("[ScalarClient] [AllTxHandler] events: %v", events)
			return nil, nil
		},
	}
	_, err := SubscribeWithTimeout(ctx, EventTimeout, network, TxEvent, internalCallback)
	// _, err := Subscribe(ctx, network, TxEvent, internalCallback)
	if err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeAllTxEvent] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeAllTxEvent] success")
	}
	return nil
}

func subscribeCommandBatchSignedEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[*types.CommandBatchSigned]) error) error {
	internalCallback := func(events []IBCEvent[*types.CommandBatchSigned]) {
		err := callback(ctx, events)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [subscribeCommandBatchSignedEvent] callback error: %v", err)
		}
	}
	_, err := SubscribeWithTimeout(ctx, EventTimeout, network, BatchCommandSignedEvent, internalCallback)
	// _, err := Subscribe(ctx, network, BatchCommandSignedEvent, internalCallback)
	if err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeCommandBatchSignedEvent] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeCommandBatchSignedEvent] success")
	}
	return nil
}
