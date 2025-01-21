package scalar

import (
	"context"

	"github.com/rs/zerolog/log"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
)

func (c *Client) handleNewBlockEvents(ctx context.Context, events map[string][]string) error {
	log.Info().Msgf("[ScalarClient] [handleNewBlockEvents] events: %d", len(events))
	//Try parse tokenSentEvent
	err := c.tryHandleTokenSentEvents(ctx, events)
	if err != nil {
		log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle token sent events: %v", err)
	}
	err = c.tryHandleMintCommandEvents(ctx, events)
	if err != nil {
		log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle mint command events: %v", err)
	}
	err = c.tryHandleContractCallWithMintApprovedEvents(ctx, events)
	if err != nil {
		log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle contract call with mint approved events: %v", err)
	}
	err = c.tryHandleContractCallApprovedEvents(ctx, events)
	if err != nil {
		log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle contract call approved events: %v", err)
	}
	err = c.tryHandleCommandBatchSignedEvent(ctx, events)
	if err != nil {
		log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle command batch signed events: %v", err)
	}
	err = c.tryHandleEVMCompletedEvent(ctx, events)
	if err != nil {
		log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle evm completed events: %v", err)
	}
	return nil
}

func (c *Client) tryHandleTokenSentEvents(ctx context.Context, events map[string][]string) error {
	tokenSentEvents, err := ParseIBCEvent[*chainstypes.EventTokenSent](events)
	if err == nil && len(tokenSentEvents) > 0 {
		log.Debug().Any("TokenSentEvents", tokenSentEvents).Msg("[ScalarClient] [handleTokenSentEvents]")
		return c.handleTokenSentEvents(ctx, tokenSentEvents)
	} else {
		log.Debug().Msg("[ScalarClient] [handleNewBlockEvents] no token sent events")
		return nil
	}
}

func (c *Client) tryHandleMintCommandEvents(ctx context.Context, events map[string][]string) error {
	mintCommandEvents, err := ParseIBCEvent[*chainstypes.MintCommand](events)
	if err == nil && len(mintCommandEvents) > 0 {
		log.Debug().Any("MintCommandEvents", mintCommandEvents).Msg("[ScalarClient] [handleMintCommandEvents]")
		return c.handleMintCommandEvents(ctx, mintCommandEvents)
	} else {
		log.Debug().Msg("[ScalarClient] [handleNewBlockEvents] no mint command events")
		return nil
	}
}

func (c *Client) tryHandleContractCallWithMintApprovedEvents(ctx context.Context, events map[string][]string) error {
	contractCallWithMintApprovedEvents, err := ParseIBCEvent[*chainstypes.EventContractCallWithMintApproved](events)
	if err == nil && len(contractCallWithMintApprovedEvents) > 0 {
		log.Debug().Any("ContractCallWithMintApprovedEvents", contractCallWithMintApprovedEvents).Msg("[ScalarClient] [handleContractCallWithMintApprovedEvents]")
		return c.handleContractCallWithMintApprovedEvents(ctx, contractCallWithMintApprovedEvents)
	} else {
		log.Debug().Msg("[ScalarClient] [handleNewBlockEvents] no contract call with mint approved events")
		return nil
	}
}
func (c *Client) tryHandleContractCallApprovedEvents(ctx context.Context, events map[string][]string) error {
	contractCallApprovedEvents, err := ParseIBCEvent[*chainstypes.ContractCallApproved](events)
	if err == nil && len(contractCallApprovedEvents) > 0 {
		log.Debug().Any("ContractCallApprovedEvents", contractCallApprovedEvents).Msg("[ScalarClient] [handleContractCallApprovedEvents]")
		return c.handleContractCallApprovedEvents(ctx, contractCallApprovedEvents)
	} else {
		log.Debug().Msg("[ScalarClient] [handleNewBlockEvents] no contract call approved events")
		return nil
	}
}

func (c *Client) tryHandleCommandBatchSignedEvent(ctx context.Context, events map[string][]string) error {
	commandBatchSignedEvents, err := ParseIBCEvent[*chainstypes.CommandBatchSigned](events)
	if err == nil && len(commandBatchSignedEvents) > 0 {
		log.Debug().Any("CommandBatchSignedEvents", commandBatchSignedEvents).Msg("[ScalarClient] [handleCommandBatchSignedEvents]")
		return c.handleCommandBatchSignedEvents(ctx, commandBatchSignedEvents)
	} else {
		log.Debug().Msg("[ScalarClient] [handleNewBlockEvents] no command batch signed events")
		return nil
	}
}

func (c *Client) tryHandleEVMCompletedEvent(ctx context.Context, events map[string][]string) error {
	evmCompletedEvents, err := ParseIBCEvent[*chainstypes.ChainEventCompleted](events)
	if err == nil && len(evmCompletedEvents) > 0 {
		log.Debug().Any("EVMCompletedEvents", evmCompletedEvents).Msg("[ScalarClient] [handleEVMCompletedEvents]")
		return c.handleCompletedEvents(ctx, evmCompletedEvents)
	} else {
		log.Debug().Msg("[ScalarClient] [handleNewBlockEvents] no evm completed events")
		return nil
	}
}
