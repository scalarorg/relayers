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
	"github.com/scalarorg/relayers/pkg/types"
)

func (c *BtcClient) handleEventBusMessage(event *events.EventEnvelope) error {
	log.Info().
		Str("eventType", event.EventType).
		Str("messageID", event.MessageID).
		Str("destinationChain", event.DestinationChain).
		Any("data", event.Data).
		Msg("[BtcClient] [handleEventBusMessage]")
	switch event.EventType {
	case events.EVENT_SCALAR_DEST_CALL_APPROVED:
		//Broadcast from scalar.handleDestCallApprovedEvent
		return c.handleScalarDestCallApproved(event.MessageID, event.Data.(string))
	case events.EVENT_CUSTODIAL_SIGNATURES_CONFIRMED:
		return c.handleCustodialSignaturesConfirmed(event.MessageID, event.Data.(string))
	}

	return nil
}

func (c *BtcClient) handleScalarDestCallApproved(messageID string, executeData string) error {
	decodedExecuteData, err := DecodeExecuteData(executeData)
	if err != nil {
		return fmt.Errorf("failed to decode execute data: %w", err)
	}
	c.observeScalarDestCallApproved(decodedExecuteData)
	if len(decodedExecuteData.Commands) != len(decodedExecuteData.Params) {
		return fmt.Errorf("[BtcClient] [handleScalarDestCallApproved] commands and params length mismatch")
	}
	for i := 0; i < len(decodedExecuteData.Commands); i++ {
		err := c.executeBtcCommand(messageID, decodedExecuteData.CommandIds[i], decodedExecuteData.Commands[i], decodedExecuteData.Params[i])
		if err != nil {
			log.Error().Msgf("[BtcClient] [handleScalarDestCallApproved] failed to execute btc command: %s, %v", decodedExecuteData.Commands[i], err)
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
func (c *BtcClient) executeBtcCommand(messageID string, commandId [32]byte, command string, params []byte) error {
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
		log.Error().Err(err).Str("payloadHash", payloadHash).Msg("[BtcClient] [executeBtcCommand] failed to find payload by hash")
		return err
	}
	//2. Extract base64 psbt from the payload
	log.Debug().Msgf("[BtcClient] [executeBtcCommand] encodedPsbtPayload %s", hex.EncodeToString(encodedPsbtPayload))

	decodedPsbtPayload, err := evm.AbiUnpack(encodedPsbtPayload, "string")
	if err != nil {
		return fmt.Errorf("[BtcClient] [executeBtcCommand] failed to abi unpack psbt: %w", err)
	}
	log.Debug().Msgf("[BtcClient] [executeBtcCommand] decodedPsbtPayload: %v", decodedPsbtPayload)

	//3. request signers for signatures
	err = c.broadcastForSignatures(messageID, executeParams, decodedPsbtPayload[0].(string))
	if err != nil {
		return fmt.Errorf("[BtcClient] [executeBtcCommand] failed to broadcast for signatures: %w", err)
	}
	return err
}
func (c *BtcClient) observeScalarDestCallApproved(decodedExecuteData *DecodedExecuteData) error {
	log.Debug().
		Uint64("chainId", decodedExecuteData.ChainId).
		Strs("commands", decodedExecuteData.Commands).
		Msg("[BtcClient] [observeScalarDestCallApproved]")
	return nil
}

func (c *BtcClient) broadcastForSignatures(messageID string, executeParams *types.ExecuteParams, base64Psbt string) error {
	//1. Detect which parties need to sign byte first 2 bytes of the base64Psbt
	//Real base64Psbt is without the first 2 bytes
	signingType, finalBase64Psbt := c.detectSigningType(executeParams, base64Psbt)
	log.Debug().Msgf("[BtcClient] [broadcastForSignatures] signingType: %v", signingType)
	var signedPsbtHex string
	var err error
	if signingType == CUSTODIAL_ONLY {
		//2. Request custodial signatures
		//Signatures will be handled by custodial network in asynchronous manner
		err = c.requestCustodialSignatures(messageID, executeParams, finalBase64Psbt)
		if err != nil {
			log.Err(err).Msg("[BtcClient] [broadcastForSignatures] failed to request custodial signaturess")
			return err
		}
	} else {
		//2. Request protocol signature
		log.Debug().Msgf("[BtcClient] [broadcastForSignatures] request protocol signature")
		signedPsbtHex, err = c.requestProtocolSignature(executeParams, finalBase64Psbt)
		if err != nil {
			return fmt.Errorf("[BtcClient] [broadcastForSignatures] failed to request protocol signature: %w", err)
		}
		//3. Broadcast to the network
		txHash, err := c.BroadcastRawTx(signedPsbtHex)
		if err != nil {
			return fmt.Errorf("[BtcClient] [broadcastForSignatures] failed to broadcast tx: %w", err)
		}
		log.Debug().Msgf("[BtcClient] [broadcastForSignatures] broadcasted txHash: %s", txHash)

	}
	//4. Todo:Update status in the db
	// err = c.dbAdapter.UpdateRelayDataStatueWithExecuteHash(messageID, relaydata.SUCCESS, txHash)
	// if err != nil {
	// 	return fmt.Errorf("[BtcClient] [broadcastForSignatures] failed to update relay data status: %w", err)
	// }
	return nil
}

func (c *BtcClient) detectSigningType(executeParams *types.ExecuteParams, base64Psbt string) (SigningType, string) {
	//To first 2 bytes of the base64Psbt, it can be used to detect the signing type
	firstTwoBytes := base64Psbt[:2]
	if firstTwoBytes == "80" {
		return USER_PROTOCOL, base64Psbt[2:]
	} else if firstTwoBytes == "40" {
		return CUSTODIAL_ONLY, base64Psbt[2:]
	} else {
		//For old format, it is signed by custodial
		return CUSTODIAL_ONLY, base64Psbt
	}
}
func (c *BtcClient) requestProtocolSignature(executeParams *types.ExecuteParams, base64Psbt string) (string, error) {
	//1. Find protocol info
	protocolInfo, err := c.dbAdapter.FindProtocolInfo(executeParams.SourceChain, executeParams.ContractAddress.Hex())
	if err != nil {
		return "", fmt.Errorf("[BtcClient] [requestProtocolSignature] failed to find protocol info by chain name and contract address: %s, %s, %w",
			executeParams.SourceChain, executeParams.ContractAddress, err)
	}
	if protocolInfo.RPCUrl == "" {
		return "", fmt.Errorf("[BtcClient] [requestProtocolSignature] protocol info does not have rpc url: %s, %s",
			executeParams.SourceChain, executeParams.ContractAddress)
	}
	//singingUrl := fmt.Sprintf("%s/v1/sign-unbonding-tx", protocolInfo.RPCUrl)
	signingUrl := protocolInfo.RPCUrl
	accessToken := protocolInfo.AccessToken
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

// Request custodial signatures from custodial network
func (c *BtcClient) requestCustodialSignatures(messageID string, executeParams *types.ExecuteParams, base64Psbt string) error {
	log.Debug().Msgf("[BtcClient] [requestCustodialSignatures] request custodial signatures")
	if c.eventBus != nil {
		c.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_BTC_SIGNATURE_REQUESTED,
			DestinationChain: events.CUSTODIAL_NETWORK_NAME,
			MessageID:        messageID,
			Data: events.SignatureRequest{
				ExecuteParams: executeParams,
				Base64Psbt:    base64Psbt,
			},
		})
	} else {
		return fmt.Errorf("[BtcClient] [requestCustodialSignatures] event bus is undefined")
	}
	//Todo: Perform custodial signing. Better version, we can handle this in the custodial network
	return nil
}

func (c *BtcClient) handleCustodialSignaturesConfirmed(messageID string, signedPsbt string) error {
	log.Debug().Msgf("[BtcClient] [handleCustodialSignaturesConfirmed] signedPsbtHex: %s", signedPsbt)
	//Broadcast to the network
	txHash, err := c.BroadcastRawTx(signedPsbt)
	if err != nil {
		log.Error().Err(err).
			Str("messageID", messageID).
			Str("signedPsbt", signedPsbt).
			Msg("[BtcClient] [handleCustodialSignaturesConfirmed] failed to broadcast tx")
		return err
	}
	txHashStr := txHash.String()
	log.Debug().Msgf("[BtcClient] [handleCustodialSignaturesConfirmed] broadcasted txHash: %s", txHash)
	err = c.dbAdapter.UpdateRelayDataStatueWithExecuteHash(messageID, relaydata.SUCCESS, &txHashStr)
	if err != nil {
		log.Error().Err(err).
			Str("messageID", messageID).
			Str("signedPsbt", signedPsbt).
			Msg("[BtcClient] [handleCustodialSignaturesConfirmed] failed to update relay data status")
	}
	return nil
}
