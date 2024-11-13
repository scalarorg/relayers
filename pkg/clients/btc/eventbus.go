package btc

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/clients/evm"
	relaydata "github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
)

func (c *BtcClient) handleEventBusMessage(event *events.EventEnvelope) error {
	log.Info().Msgf("[BtcClient] [handleEventBusMessage] event: %v", event)
	switch event.EventType {
	case events.EVENT_SCALAR_CONTRACT_CALL_APPROVED:
		//Broadcast from scalar.handleContractCallApprovedEvent
		return c.handleScalarContractCallApproved(event.MessageID, event.Data.(string))

	}
	return nil
}

func (c *BtcClient) handleScalarContractCallApproved(messageID string, executeData string) error {
	decodedExecuteData, err := DecodeExecuteData(executeData)
	if err != nil {
		return fmt.Errorf("failed to decode execute data: %w", err)
	}
	c.observeScalarContractCallApproved(decodedExecuteData)
	if len(decodedExecuteData.Commands) != len(decodedExecuteData.Params) {
		return fmt.Errorf("[BtcClient] [handleScalarContractCallApproved] commands and params length mismatch")
	}
	for i := 0; i < len(decodedExecuteData.Commands); i++ {
		err := c.executeBtcCommand(decodedExecuteData.CommandIds[i], decodedExecuteData.Commands[i], decodedExecuteData.Params[i])
		if err != nil {
			log.Error().Msgf("[BtcClient] [handleScalarContractCallApproved] failed to execute btc command: %s, %v", decodedExecuteData.Commands[i], err)
			return fmt.Errorf("failed to execute btc command: %w", err)
		}
	}
	//2. Update status of the event
	//txHash := tx.Hash().String()
	err = c.dbAdapter.UpdateRelayDataStatueWithExecuteHash(messageID, relaydata.SUCCESS, nil)
	if err != nil {
		return fmt.Errorf("failed to update relay data status: %w", err)
	}
	return nil
}
func (c *BtcClient) executeBtcCommand(commandId [32]byte, command string, params []byte) error {
	executeParams, err := ParseExecuteParams(params)
	if err != nil {
		return fmt.Errorf("[BtcClient] [executeBtcCommand] failed to parse execute params: %w", err)
	}
	log.Debug().Msgf("[BtcClient] [executeBtcCommand] executeParams: %v", executeParams)
	//1. Find payload by hash from db
	// This payload is stored in the db when the VaultTx is indexed
	payloadHash := hex.EncodeToString(executeParams.PayloadHash[:])
	encodedPsbtPayload, err := c.dbAdapter.FindPayloadByHash(payloadHash)
	if err != nil {
		return fmt.Errorf("[BtcClient] [executeBtcCommand] failed to find payload by hash %s: %w", payloadHash, err)
	}
	//2. Extract base64 psbt from the payload
	decodedPsbtPayload, err := evm.AbiUnpack(encodedPsbtPayload)
	if err != nil {
		return fmt.Errorf("[BtcClient] [executeBtcCommand] failed to abi unpack psbt: %w", err)
	}
	log.Debug().Msgf("[BtcClient] [executeBtcCommand] decodedPsbtPayload: %v", decodedPsbtPayload)

	//3. request signers for signatures
	c.broadcastForSignatures(executeParams, decodedPsbtPayload[0].(string))
	return nil
}
func (c *BtcClient) observeScalarContractCallApproved(decodedExecuteData *DecodedExecuteData) error {
	log.Debug().Msgf("[BtcClient] [observeScalarContractCallApproved] decodedExecuteData: %v", decodedExecuteData)
	return nil
}

func (c *BtcClient) broadcastForSignatures(executeParams *ExecuteParams, base64Psbt string) error {
	//Todo. Check wich parties need to sign then broadcast to them
	//First version, we get protocol signer from db and broadcast to it
	//Add token address to the query when we support multiple tokens
	//1. Find protocol info
	protocolInfo, err := c.dbAdapter.FindProtocolInfo(executeParams.SourceChain, executeParams.ContractAddress)
	if err != nil {
		return fmt.Errorf("[BtcClient] [broadcastForSignatures] failed to find protocol info by chain name and contract address: %s, %s, %w",
			executeParams.SourceChain, executeParams.ContractAddress, err)
	}
	if protocolInfo.RPCUrl == "" {
		return fmt.Errorf("[BtcClient] [broadcastForSignatures] protocol info does not have rpc url: %s, %s",
			executeParams.SourceChain, executeParams.ContractAddress)
	}
	singingUrl := fmt.Sprintf("%s/v1/sign-unbonding-tx", protocolInfo.RPCUrl)
	accessToken := protocolInfo.AccessToken
	//2.Broadcast to the rpc url
	signedPsbtHex, err := c.requestProtocolSignature(singingUrl, accessToken, executeParams, base64Psbt)
	if err != nil {
		return fmt.Errorf("[BtcClient] [broadcastForSignatures] failed to request protocol signature: %w", err)
	}
	log.Debug().Msgf("[BtcClient] [broadcastForSignatures] signedPsbtHex: %s", signedPsbtHex)
	//3. Broadcast to the network
	maxFeeRate := 0.10
	txHash, err := c.BroadcastRawTx(signedPsbtHex, &maxFeeRate)
	if err != nil {
		return fmt.Errorf("[BtcClient] [broadcastForSignatures] failed to broadcast tx: %w", err)
	}
	log.Debug().Msgf("[BtcClient] [broadcastForSignatures] broadcasted txHash: %s", txHash)
	//4. Todo:Update status in the db
	// err = c.dbAdapter.UpdateRelayDataStatueWithExecuteHash(messageID, relaydata.SUCCESS, txHash)
	// if err != nil {
	// 	return fmt.Errorf("[BtcClient] [broadcastForSignatures] failed to update relay data status: %w", err)
	// }
	return nil
}

func (c *BtcClient) requestProtocolSignature(signingUrl string, accessToken string, executeParams *ExecuteParams, base64Psbt string) (string, error) {
	// Create request payload
	payload := map[string]interface{}{
		"evm_chain_name":        executeParams.SourceChain,
		"evm_tx_id":             executeParams.SourceTxHash,
		"unbonding_psbt_base64": base64Psbt,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request payload: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", signingUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make POST request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	//Todo: Modify protocol signer to return signed psbt hex only
	var response struct {
		TxId  string `json:"tx_id"`  //If protocol signer broadcast the it return broadcasted Tx
		TxHex string `json:"tx_hex"` //Signed raw btc tx hex, which is ready for broadcast
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	if response.TxHex == "" {
		return "", fmt.Errorf("Signed psbt not found: %s", response.TxHex)
	}
	return response.TxHex, nil
}
