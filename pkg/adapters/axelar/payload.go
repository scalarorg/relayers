package axelar

// import (
// 	"encoding/hex"
// 	"fmt"
// 	"strings"

// 	sdk "github.com/cosmos/cosmos-sdk/types"
// )

// // GetConfirmGatewayTxPayload creates a payload for confirming gateway transactions on EVM chain
// func GetConfirmGatewayTxPayload(sender string, chain string, txHash string) ([]sdk.Msg, error) {
// 	// Convert sender to account address
// 	senderAddr, err := sdk.AccAddressFromBech32(sender)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid sender address: %w", err)
// 	}

// 	// Remove "0x" prefix if present
// 	txHash = strings.TrimPrefix(txHash, "0x")

// 	// Convert txHash to bytes
// 	txBytes, err := hex.DecodeString(txHash)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid transaction hash: %w", err)
// 	}

// 	msg := &ConfirmGatewayTxRequest{
// 		Sender: senderAddr.Bytes(),
// 		Chain:  chain,
// 		TxID:   txBytes,
// 	}

// 	return []sdk.Msg{
// 		sdk.NewMsgWithTypeURL(
// 			fmt.Sprintf("/%s.ConfirmGatewayTxRequest", EvmProtobufPackage),
// 			msg,
// 		),
// 	}, nil
// }

// // GetCallContractRequest creates a payload for contract calls
// func GetCallContractRequest(sender string, chain string, contractAddress string, payload string) ([]sdk.Msg, error) {
// 	senderAddr, err := sdk.AccAddressFromBech32(sender)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid sender address: %w", err)
// 	}

// 	// Remove "0x" prefix if present
// 	payload = strings.TrimPrefix(payload, "0x")

// 	// Convert payload to bytes
// 	payloadBytes, err := hex.DecodeString(payload)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid payload: %w", err)
// 	}

// 	msg := &CallContractRequest{
// 		Sender:          senderAddr.Bytes(),
// 		Chain:           chain,
// 		ContractAddress: contractAddress,
// 		Payload:         payloadBytes,
// 	}

// 	return []sdk.Msg{
// 		sdk.NewMsgWithTypeURL(
// 			fmt.Sprintf("/%s.CallContractRequest", AxelarProtobufPackage),
// 			msg,
// 		),
// 	}, nil
// }

// // GetRouteMessageRequest creates a payload for routing messages
// func GetRouteMessageRequest(sender string, txHash string, logIndex int, payload string) ([]sdk.Msg, error) {
// 	senderAddr, err := sdk.AccAddressFromBech32(sender)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid sender address: %w", err)
// 	}

// 	// Remove "0x" prefix if present
// 	payload = strings.TrimPrefix(payload, "0x")

// 	// Convert payload to bytes
// 	payloadBytes, err := hex.DecodeString(payload)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid payload: %w", err)
// 	}

// 	// Generate ID based on txHash and logIndex
// 	var id string
// 	if logIndex == -1 {
// 		id = txHash
// 	} else {
// 		id = fmt.Sprintf("%s-%d", txHash, logIndex)
// 	}

// 	msg := &RouteMessageRequest{
// 		Sender:  senderAddr.Bytes(),
// 		Payload: payloadBytes,
// 		ID:      id,
// 	}

// 	return []sdk.Msg{
// 		sdk.NewMsgWithTypeURL(
// 			fmt.Sprintf("/%s.RouteMessageRequest", AxelarProtobufPackage),
// 			msg,
// 		),
// 	}, nil
// }

// // GetSignCommandPayload creates a payload for signing commands
// func GetSignCommandPayload(sender string, chain string) ([]sdk.Msg, error) {
// 	senderAddr, err := sdk.AccAddressFromBech32(sender)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid sender address: %w", err)
// 	}

// 	msg := &SignCommandsRequest{
// 		Sender: senderAddr.Bytes(),
// 		Chain:  chain,
// 	}

// 	return []sdk.Msg{
// 		sdk.NewMsgWithTypeURL(
// 			fmt.Sprintf("/%s.SignCommandsRequest", EvmProtobufPackage),
// 			msg,
// 		),
// 	}, nil
// }
