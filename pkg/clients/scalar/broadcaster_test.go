package scalar_test

import (
	"context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	chainsExported "github.com/scalarorg/scalar-core/x/chains/exported"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	"github.com/stretchr/testify/require"
)

func TestEnqueueMessage(t *testing.T) {
	pendingCommands := scalar.NewPendingCommands()
	broadcaster := scalar.NewBroadcaster(nil, pendingCommands, time.Second, 1)
	hash := chainsExported.Hash(common.HexToHash("f0510bcacb2e428bd89e39e9708555265ed413b5320c5f920bf4becac9c53f56"))
	commandRequest := chainstypes.NewConfirmSourceTxsRequest(
		sdk.AccAddress{},
		"sepolia",
		[]chainsExported.Hash{hash},
	)
	err := broadcaster.Start(context.Background())
	require.NoError(t, err)
	err = broadcaster.QueueTxMsg(commandRequest)
	require.NoError(t, err)
	time.Sleep(10 * time.Second)
}
