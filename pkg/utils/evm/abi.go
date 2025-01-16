package evm

//Todo: Move to bitcoin-vault package
import (
	"github.com/ethereum/go-ethereum/accounts/abi"
)

// func CalculateStakingPayload(vaultTx *types.VaultTransaction) ([]byte, string, error) {
// 	toAddress, err := hex.DecodeString(vaultTx.DestRecipientAddress)
// 	if err != nil {
// 		return nil, "", fmt.Errorf("failed to decode toAddress: %w", err)
// 	}
// 	// var toAddressBytes [20]byte
// 	// copy(toAddressBytes[:], toAddress)

// 	txHash := strings.TrimPrefix(vaultTx.TxHash, "0x")
// 	// Convert hex string to [32]byte
// 	// txHash, err := hex.DecodeString(strings.TrimPrefix(vaultTx.TxHash, "0x"))
// 	// if err != nil {
// 	// 	return nil, "", fmt.Errorf("failed to decode txHash: %w", err)
// 	// }
// 	// var txHashBytes [32]byte
// 	// copy(txHashBytes[:], txHash)
// 	log.Debug().Str("TxHash", txHash).Uint64("Amount", vaultTx.Amount).Str("RecipientAddress", hex.EncodeToString(toAddress)).Msg("CalculateDestPayload")
// 	payloadBytes, payloadHash, err := encode.SafeCalculateDestPayload(vaultTx.Amount, txHash, toAddress)
// 	if err != nil {
// 		return nil, "", fmt.Errorf("failed to calculate payload: %w", err)
// 	}
// 	return payloadBytes, hex.EncodeToString(payloadHash), nil
// }

// // Calculate payload and payload hash
// // Todo: implement this in the bitcoin-vault lib then call it via ffi or in some pure golang code
// func CalculatePayload(vaultTx *types.VaultTransaction) ([]byte, string, error) {
// 	toAddress := strings.ToLower(vaultTx.DestRecipientAddress)
// 	amount := vaultTx.Amount
// 	txHash := strings.TrimPrefix(vaultTx.TxHash, "0x")
// 	// Convert hex string to [32]byte
// 	txHashBytes, err := hex.DecodeString(txHash)
// 	if err != nil {
// 		return nil, "", fmt.Errorf("failed to decode txHash: %w", err)
// 	}
// 	var txHashArray [32]byte
// 	copy(txHashArray[:], txHashBytes)

// 	// Create arguments array
// 	arguments := abi.Arguments{
// 		{Type: GetAddressType()},
// 		{Type: GetUint256Type()},
// 		{Type: GetBytes32Type()},
// 	}

// 	// Pack the values
// 	payloadBytes, err := arguments.Pack(
// 		common.HexToAddress(toAddress),
// 		new(big.Int).SetUint64(amount),
// 		txHashArray,
// 	)
// 	if err != nil {
// 		return nil, "", fmt.Errorf("failed to pack values: %w", err)
// 	}
// 	payloadHash := crypto.Keccak256(payloadBytes)
// 	return payloadBytes, hex.EncodeToString(payloadHash), nil
// }

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
