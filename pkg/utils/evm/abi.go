package evm

//Todo: Move to bitcoin-vault package
import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	evmutils "github.com/scalarorg/bitcoin-vault/go-utils/evm"
	"github.com/scalarorg/go-electrum/electrum/types"
)

func CalculateStakingPayload(vaultTx *types.VaultTransaction) ([]byte, string, error) {
	toAddress, err := hex.DecodeString(vaultTx.DestRecipientAddress)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode toAddress: %w", err)
	}
	var toAddressBytes [20]byte
	copy(toAddressBytes[:], toAddress)

	// Convert hex string to [32]byte
	txHash, err := hex.DecodeString(strings.TrimPrefix(vaultTx.TxHash, "0x"))
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode txHash: %w", err)
	}
	var txHashBytes [32]byte
	copy(txHashBytes[:], txHash)
	payloadBytes, payloadHash, err := evmutils.CalculateStakingPayloadHash(toAddressBytes, int64(vaultTx.Amount), txHashBytes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to calculate payload: %w", err)
	}
	return payloadBytes, hex.EncodeToString(payloadHash), nil
}

// Calculate payload and payload hash
// Todo: implement this in the bitcoin-vault lib then call it via ffi or in some pure golang code
func CalculatePayload(vaultTx *types.VaultTransaction) ([]byte, string, error) {
	toAddress := strings.ToLower(vaultTx.DestRecipientAddress)
	amount := vaultTx.Amount
	txHash := strings.TrimPrefix(vaultTx.TxHash, "0x")
	// Convert hex string to [32]byte
	txHashBytes, err := hex.DecodeString(txHash)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode txHash: %w", err)
	}
	var txHashArray [32]byte
	copy(txHashArray[:], txHashBytes)

	// Create arguments array
	arguments := abi.Arguments{
		{Type: GetAddressType()},
		{Type: GetUint256Type()},
		{Type: GetBytes32Type()},
	}

	// Pack the values
	payloadBytes, err := arguments.Pack(
		common.HexToAddress(toAddress),
		new(big.Int).SetUint64(amount),
		txHashArray,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to pack values: %w", err)
	}
	payloadHash := crypto.Keccak256(payloadBytes)
	return payloadBytes, hex.EncodeToString(payloadHash), nil
}

// Helper functions to create ABI types
func GetAddressType() abi.Type {
	t, _ := abi.NewType("address", "", nil)
	return t
}

func GetUint256Type() abi.Type {
	t, _ := abi.NewType("uint256", "", nil)
	return t
}

func GetBytes32Type() abi.Type {
	t, _ := abi.NewType("bytes32", "", nil)
	return t
}
