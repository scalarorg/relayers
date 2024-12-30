package evm_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/stretchr/testify/assert"
)

func TestExecuteDestinationCall(t *testing.T) {
	event := contracts.IScalarGatewayContractCallApproved{
		CommandId:       [32]byte{165, 197, 245, 62, 7, 21, 210, 155, 218, 25, 18, 193, 83, 219, 37, 131, 103, 126, 138, 174, 215, 52, 133, 121, 246, 176, 19, 134, 187, 157, 203, 86},
		SourceChain:     "bitcoin|4",
		SourceAddress:   "tb1q2rwweg2c48y8966qt4fzj0f4zyg9wty7tykzwg",
		ContractAddress: common.HexToAddress("0x6e3B806C5F6413e0a0670666301ccB6b10628A52"),
	}
	// evmConfig.PrivateKey = "aff969005325629eee2292e44b66aef1ec26426de7aecde2dc55b85648b84a91"
	// auth, err := evm.CreateEvmAuth(evmConfig)
	// assert.NoError(t, err)
	// evmClient.SetAuth(auth)
	payload, err := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000003e8324e4d7268b0b65ba6fd71359b6460a82a9fae17af7434b48ad632e203156b750000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000001424a1db57fa3ecafcbad91d6ef068439aceeae090000000000000000000000000")
	assert.NoError(t, err)
	receipt, err := evmClient.ExecuteDestinationCall(event.ContractAddress, event.CommandId, event.SourceChain, event.SourceAddress, payload)
	assert.NoError(t, err)
	fmt.Printf("Receipt %v", receipt)
}
