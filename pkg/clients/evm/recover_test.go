package evm_test

import (
	"context"
	"testing"

	"github.com/scalarorg/relayers/pkg/events"
)

func TestRecoverEvents(t *testing.T) {
	eventNames := []string{
		events.EVENT_EVM_CONTRACT_CALL,
		events.EVENT_EVM_CONTRACT_CALL_APPROVED,
		events.EVENT_EVM_TOKEN_SENT,
		events.EVENT_EVM_CONTRACT_CALL_APPROVED,
		events.EVENT_EVM_COMMAND_EXECUTED,
	}
	go func() {
		evmClient.ProcessMissingLogs()
	}()
	evmClient.RecoverEvents(context.Background(), eventNames)
	select {}
}
