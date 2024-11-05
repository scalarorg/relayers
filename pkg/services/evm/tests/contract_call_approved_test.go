package evm_test

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestContractCallApprovedListener(t *testing.T) {
	TEST_RPC_ENDPOINT := "https://eth-sepolia.g.alchemy.com/v2/nNbspp-yjKP9GtAcdKi8xcLnBTptR2Zx"
	TEST_CONTRACT_ADDRESS := "0x2bb588d7bb6faAA93f656C3C78fFc1bEAfd1813D"

	// Setup
	ctx, cancel := context.WithTimeout(context.Background(), 29*time.Minute)
	defer cancel()

	// Connect to a test network

	rpc, err := rpc.DialContext(ctx, TEST_RPC_ENDPOINT)
	require.NoError(t, err)
	defer rpc.Close()

	client := ethclient.NewClient(rpc)
	require.NoError(t, err)
	defer client.Close()

	// Initialize the contract
	contractAddress := common.HexToAddress(TEST_CONTRACT_ADDRESS)
	require.NoError(t, err)

	// Add ABI parsing
	contractAbi := `[
		{
			"type": "event",
			"name": "ContractCallApproved",
			"inputs": [
				{"indexed": true, "name": "commandId", "type": "bytes32"},
				{"indexed": false, "name": "sourceChain", "type": "string"},
				{"indexed": false, "name": "sourceAddress", "type": "string"},
				{"indexed": true, "name": "contractAddress", "type": "address"},
				{"indexed": true, "name": "payloadHash", "type": "bytes32"},
				{"indexed": false, "name": "sourceTxHash", "type": "bytes32"},
				{"indexed": false, "name": "sourceEventIndex", "type": "uint256"}
			]
		}
	]`

	parsedAbi, err := abi.JSON(strings.NewReader(contractAbi))
	require.NoError(t, err)

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(6922244),
		ToBlock:   big.NewInt(6922244),
		Addresses: []common.Address{
			contractAddress,
		},
	}

	logs, err := client.FilterLogs(context.Background(), query)
	require.NoError(t, err)

	fmt.Println(len(logs))

	for idx, log := range logs {
		fmt.Printf("-----------------------------------\n")

		t.Logf("Log: %+v", log.TxHash)

		event := struct {
			CommandId        [32]byte
			SourceChain      string
			SourceAddress    string
			ContractAddress  common.Address
			PayloadHash      [32]byte
			SourceTxHash     [32]byte
			SourceEventIndex *big.Int
		}{}

		if err := parsedAbi.UnpackIntoInterface(&event, "ContractCallApproved", log.Data); err != nil {
			t.Logf("Failed to parse log: %d", idx)
			continue
		}

		t.Logf("Event details:\n"+
			"CommandId: %x\n"+
			"SourceChain: %s\n"+
			"SourceAddress: %s\n"+
			"ContractAddress: %s\n"+
			"PayloadHash: %x\n"+
			"SourceTxHash: %x\n"+
			"SourceEventIndex: %s",
			event.CommandId,
			event.SourceChain,
			event.SourceAddress,
			event.ContractAddress.Hex(),
			event.PayloadHash,
			event.SourceTxHash,
			event.SourceEventIndex.String())
	}

	// gateway, err := contracts.NewIAxelarGateway(contractAddress, client)

}
