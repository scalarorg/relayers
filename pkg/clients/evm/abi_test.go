package evm_test

import (
	"fmt"
	"testing"

	"github.com/scalarorg/relayers/pkg/clients/evm"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/stretchr/testify/require"
)

func TestGetEventByName(t *testing.T) {
	redeemEvent, ok := evm.GetEventByName(events.EVENT_EVM_REDEEM_TOKEN)
	require.True(t, ok)
	require.NotNil(t, redeemEvent)
	fmt.Printf("redeemEvent %v\n", redeemEvent)
}
