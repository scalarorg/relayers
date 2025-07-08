package scalar

import (
	"context"

	"github.com/rs/zerolog/log"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	covTypes "github.com/scalarorg/scalar-core/x/covenant/types"
)

// TODO: refactor this function to use a more efficient way to parse the events
func (c *Client) handleNewBlockEvents(ctx context.Context, events map[string][]string) error {
	log.Info().Msgf("[ScalarClient] [handleNewBlockEvents] events: %d", len(events))
	parsedEvents, err := ParseBlockEvents(events)
	if err != nil {
		log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to parse block events: %v", err)
	} else {
		log.Info().Int("TokenSentEvents", len(parsedEvents.tokenSentEvents)).
			Int("MintCommandEvents", len(parsedEvents.mintCommandEvents)).
			Int("ContractCallWithMintApprovedEvents", len(parsedEvents.contractCallWithMintApprovedEvents)).
			Int("ContractCallApprovedEvents", len(parsedEvents.contractCallApprovedEvents)).
			Int("RedeemTokenApprovedEvents", len(parsedEvents.redeemTokenApprovedEvents)).
			Int("CommandBatchSignedEvents", len(parsedEvents.commandBatchSignedEvents)).
			Int("EVMCompletedEvents", len(parsedEvents.evmCompletedEvents)).
			Int("SwitchPhaseStartedEvents", len(parsedEvents.switchPhaseStartedEvents)).
			Int("SwitchPhaseCompletedEvents", len(parsedEvents.switchPhaseCompletedEvents)).
			Msg("[ScalarClient] [handleNewBlockEvents] parsed events")
	}
	if len(parsedEvents.tokenSentEvents) > 0 {
		err = c.handleTokenSentEvents(ctx, parsedEvents.tokenSentEvents)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle token sent events: %v", err)
		} else {
			log.Debug().Msgf("[ScalarClient] [handleNewBlockEvents] successfully handled %d token sent events", len(parsedEvents.tokenSentEvents))
		}
	}
	if len(parsedEvents.mintCommandEvents) > 0 {
		err = c.handleMintCommandEvents(ctx, parsedEvents.mintCommandEvents)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle mint command events: %v", err)
		} else {
			log.Debug().Msgf("[ScalarClient] [handleNewBlockEvents] successfully handled %d mint command events", len(parsedEvents.mintCommandEvents))
		}
	}
	if len(parsedEvents.contractCallWithMintApprovedEvents) > 0 {
		err = c.handleContractCallWithMintApprovedEvents(ctx, parsedEvents.contractCallWithMintApprovedEvents)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle contract call with mint approved events: %v", err)
		} else {
			log.Debug().Msgf("[ScalarClient] [handleNewBlockEvents] successfully handled %d contract call with mint approved events", len(parsedEvents.contractCallWithMintApprovedEvents))
		}
	}
	if len(parsedEvents.contractCallApprovedEvents) > 0 {
		err = c.handleContractCallApprovedEvents(ctx, parsedEvents.contractCallApprovedEvents)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle contract call approved events: %v", err)
		} else {
			log.Debug().Msgf("[ScalarClient] [handleNewBlockEvents] successfully handled %d contract call approved events", len(parsedEvents.contractCallApprovedEvents))
		}
	}
	if len(parsedEvents.redeemTokenApprovedEvents) > 0 {
		err = c.handleRedeemTokenApprovedEvents(ctx, parsedEvents.redeemTokenApprovedEvents)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle redeem token approved events: %v", err)
		} else {
			log.Debug().Msgf("[ScalarClient] [handleNewBlockEvents] successfully handled %d redeem token approved events", len(parsedEvents.redeemTokenApprovedEvents))
		}
	}
	if len(parsedEvents.commandBatchSignedEvents) > 0 {
		err = c.handleCommandBatchSignedEvents(ctx, parsedEvents.commandBatchSignedEvents)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle command batch signed events: %v", err)
		} else {
			log.Debug().Msgf("[ScalarClient] [handleNewBlockEvents] successfully handled %d command batch signed events", len(parsedEvents.commandBatchSignedEvents))
		}
	}
	if len(parsedEvents.evmCompletedEvents) > 0 {
		err = c.handleCompletedEvents(ctx, parsedEvents.evmCompletedEvents)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle evm completed events: %v", err)
		} else {
			log.Debug().Msgf("[ScalarClient] [handleNewBlockEvents] successfully handled %d evm completed events", len(parsedEvents.evmCompletedEvents))
		}
	}
	if len(parsedEvents.switchPhaseStartedEvents) > 0 {
		err = c.handleSwitchPhaseStartedEvents(ctx, parsedEvents.switchPhaseStartedEvents)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle switch phase started events: %v", err)
		} else {
			log.Debug().Msgf("[ScalarClient] [handleNewBlockEvents] successfully handled %d switch phase started events", len(parsedEvents.switchPhaseStartedEvents))
		}
	}
	if len(parsedEvents.switchPhaseCompletedEvents) > 0 {
		err = c.handleSwitchPhaseCompletedEvents(ctx, parsedEvents.switchPhaseCompletedEvents)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle switch phase completed events: %v", err)
		} else {
			log.Debug().Msgf("[ScalarClient] [handleNewBlockEvents] successfully handled %d switch phase completed events", len(parsedEvents.switchPhaseCompletedEvents))
		}
	}
	// Try parse tokenSentEvent
	// err = c.tryHandleTokenSentEvents(ctx, events)
	// if err != nil {
	// 	log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle token sent events: %v", err)
	// }
	// err = c.tryHandleMintCommandEvents(ctx, events)
	// if err != nil {
	// 	log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle mint command events: %v", err)
	// }
	// err = c.tryHandleContractCallWithMintApprovedEvents(ctx, events)
	// if err != nil {
	// 	log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle contract call with mint approved events: %v", err)
	// }
	// err = c.tryHandleContractCallApprovedEvents(ctx, events)
	// if err != nil {
	// 	log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle contract call approved events: %v", err)
	// }
	// err = c.tryHandleRedeemTokenApprovedEvents(ctx, events)
	// if err != nil {
	// 	log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle redeem token approved events: %v", err)
	// }
	// err = c.tryHandleCommandBatchSignedEvent(ctx, events)
	// if err != nil {
	// 	log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle command batch signed events: %v", err)
	// }
	// err = c.tryHandleEVMCompletedEvent(ctx, events)
	// if err != nil {
	// 	log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle evm completed events: %v", err)
	// }
	// err = c.tryHandleSwitchPhaseStartedEvents(ctx, events)
	// if err != nil {
	// 	log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle btc block events: %v", err)
	// }
	// err = c.tryHandleSwitchPhaseCompletedEvents(ctx, events)
	// if err != nil {
	// 	log.Error().Msgf("[ScalarClient] [handleNewBlockEvents] failed to handle switch phase completed events: %v", err)
	// }
	return nil
}

func (c *Client) tryHandleTokenSentEvents(ctx context.Context, events map[string][]string) error {
	ibcEvents, err := ParseIBCEvent[*chainstypes.EventTokenSent](events)
	if err == nil && len(events) > 0 {
		log.Debug().Int("Number of TokenSentEvents", len(events)).Msg("[ScalarClient] [handleTokenSentEvents]")
		tokenSentEvents := make([]*chainstypes.EventTokenSent, len(ibcEvents))
		for i, event := range ibcEvents {
			tokenSentEvents[i] = event.Args
			log.Debug().Any("TokenSentEvent", tokenSentEvents[i]).Msg("[ScalarClient] [handleTokenSentEvents]")
		}
		return c.handleTokenSentEvents(ctx, tokenSentEvents)
	} else {
		log.Debug().Msg("[ScalarClient] [handleNewBlockEvents] no token sent events")
		return nil
	}
}

func (c *Client) tryHandleMintCommandEvents(ctx context.Context, events map[string][]string) error {
	ibcEvents, err := ParseIBCEvent[*chainstypes.MintCommand](events)
	if err == nil && len(ibcEvents) > 0 {
		log.Debug().Any("MintCommandEvents", ibcEvents).Msg("[ScalarClient] [handleMintCommandEvents]")
		mintCommandEvents := make([]*chainstypes.MintCommand, len(ibcEvents))
		for i, event := range ibcEvents {
			mintCommandEvents[i] = event.Args
		}
		return c.handleMintCommandEvents(ctx, mintCommandEvents)
	} else {
		log.Debug().Msg("[ScalarClient] [handleNewBlockEvents] no mint command events")
		return nil
	}
}

func (c *Client) tryHandleContractCallWithMintApprovedEvents(ctx context.Context, events map[string][]string) error {
	ibcEvents, err := ParseIBCEvent[*chainstypes.EventContractCallWithMintApproved](events)
	if err == nil && len(ibcEvents) > 0 {
		log.Debug().Any("ContractCallWithMintApprovedEvents", ibcEvents).Msg("[ScalarClient] [handleContractCallWithMintApprovedEvents]")
		contractCallWithMintApprovedEvents := make([]*chainstypes.EventContractCallWithMintApproved, len(ibcEvents))
		for i, event := range ibcEvents {
			contractCallWithMintApprovedEvents[i] = event.Args
		}
		return c.handleContractCallWithMintApprovedEvents(ctx, contractCallWithMintApprovedEvents)
	} else {
		log.Debug().Msg("[ScalarClient] [handleNewBlockEvents] no contract call with mint approved events")
		return nil
	}
}
func (c *Client) tryHandleContractCallApprovedEvents(ctx context.Context, events map[string][]string) error {
	ibcEvents, err := ParseIBCEvent[*chainstypes.ContractCallApproved](events)
	if err == nil && len(ibcEvents) > 0 {
		log.Debug().Any("ContractCallApprovedEvents", ibcEvents).Msg("[ScalarClient] [handleContractCallApprovedEvents]")
		contractCallApprovedEvents := make([]*chainstypes.ContractCallApproved, len(ibcEvents))
		for i, event := range ibcEvents {
			contractCallApprovedEvents[i] = event.Args
		}
		return c.handleContractCallApprovedEvents(ctx, contractCallApprovedEvents)
	} else {
		log.Debug().Msg("[ScalarClient] [handleNewBlockEvents] no contract call approved events")
		return nil
	}
}

func (c *Client) tryHandleRedeemTokenApprovedEvents(ctx context.Context, events map[string][]string) error {
	ibcEvents, err := ParseIBCEvent[*chainstypes.EventRedeemTokenApproved](events)
	if err == nil && len(ibcEvents) > 0 {
		log.Debug().Any("RedeemTokenApprovedEvents", ibcEvents).Msg("[ScalarClient] [tryHandleRedeemTokenApprovedEvents]")
		redeemTokenApprovedEvents := make([]*chainstypes.EventRedeemTokenApproved, len(ibcEvents))
		for i, event := range ibcEvents {
			redeemTokenApprovedEvents[i] = event.Args
		}
		return c.handleRedeemTokenApprovedEvents(ctx, redeemTokenApprovedEvents)
	} else {
		log.Debug().Msg("[ScalarClient] [handleNewBlockEvents] no redeem token approved events")
		return nil
	}
}

func (c *Client) tryHandleCommandBatchSignedEvent(ctx context.Context, events map[string][]string) error {
	ibcEvents, err := ParseIBCEvent[*chainstypes.CommandBatchSigned](events)
	if err == nil && len(ibcEvents) > 0 {
		log.Debug().Any("CommandBatchSignedEvents", ibcEvents).Msg("[ScalarClient] [handleCommandBatchSignedEvents]")
		commandBatchSignedEvents := make([]*chainstypes.CommandBatchSigned, len(ibcEvents))
		for i, event := range ibcEvents {
			commandBatchSignedEvents[i] = event.Args
		}
		return c.handleCommandBatchSignedEvents(ctx, commandBatchSignedEvents)
	} else {
		log.Debug().Msg("[ScalarClient] [handleNewBlockEvents] no command batch signed events")
		return nil
	}
}

func (c *Client) tryHandleEVMCompletedEvent(ctx context.Context, events map[string][]string) error {
	ibcEvents, err := ParseIBCEvent[*chainstypes.ChainEventCompleted](events)
	if err == nil && len(ibcEvents) > 0 {
		log.Debug().Any("EVMCompletedEvents", ibcEvents).Msg("[ScalarClient] [handleEVMCompletedEvents]")
		evmCompletedEvents := make([]*chainstypes.ChainEventCompleted, len(ibcEvents))
		for i, event := range ibcEvents {
			evmCompletedEvents[i] = event.Args
		}
		return c.handleCompletedEvents(ctx, evmCompletedEvents)
	} else {
		log.Debug().Msg("[ScalarClient] [handleNewBlockEvents] no evm completed events")
		return nil
	}
}

// Parse the started switch phase event, each event is for a evm chain
func (c *Client) tryHandleSwitchPhaseStartedEvents(ctx context.Context, events map[string][]string) error {
	ibcEvents, err := ParseIBCEvent[*covTypes.SwitchPhaseStarted](events)
	if err == nil && len(ibcEvents) > 0 {
		log.Debug().Any("SwitchPhaseStartedEvents", ibcEvents).Msg("[ScalarClient] [tryHandleSwitchPhaseStartedEvents]")
		switchPhaseStartedEvents := make([]*covTypes.SwitchPhaseStarted, len(ibcEvents))
		for i, event := range ibcEvents {
			switchPhaseStartedEvents[i] = event.Args
		}
		return c.handleSwitchPhaseStartedEvents(ctx, switchPhaseStartedEvents)
	} else {
		log.Debug().Msg("[ScalarClient] [handleNewBlockEvents] no started switch phase events")
		return nil
	}
}

// Parse the started switch phase event, each event is for a evm chain
func (c *Client) tryHandleSwitchPhaseCompletedEvents(ctx context.Context, events map[string][]string) error {
	ibcEvents, err := ParseIBCEvent[*covTypes.SwitchPhaseCompleted](events)
	if err == nil && len(ibcEvents) > 0 {
		log.Debug().Any("SwitchPhaseCompletedEvents", ibcEvents).Msg("[ScalarClient] [tryHandleSwitchPhaseCompletedEvents]")
		switchPhaseCompletedEvents := make([]*covTypes.SwitchPhaseCompleted, len(ibcEvents))
		for i, event := range ibcEvents {
			switchPhaseCompletedEvents[i] = event.Args
		}
		return c.handleSwitchPhaseCompletedEvents(ctx, switchPhaseCompletedEvents)
	} else {
		log.Debug().Msg("[ScalarClient] [handleNewBlockEvents] no switch phase completed events")
		return nil
	}
}
