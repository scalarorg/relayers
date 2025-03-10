package custodial

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/wire"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/events"
)

func (c *Client) handleEventBusMessage(event *events.EventEnvelope) error {
	log.Info().Msgf("[CustodialClient] [handleEventBusMessage]: %v", event)
	switch event.EventType {
	case events.EVENT_BTC_SIGNATURE_REQUESTED:
		err := c.handleBtcSignatureRequested(event.MessageID, event.Data.(events.SignatureRequest))
		if err != nil {
			log.Error().Err(err).Msg("[CustodialClient] [handleEventBusMessage] failed to handle btc signature requested event")
			return err
		}
	}
	return nil
}

func (c *Client) handleBtcSignatureRequested(messageID string, signatureRequest events.SignatureRequest) error {
	log.Debug().Msgf("[CustodialClient] [handleBtcSignatureRequested] signatureRequest: %v", signatureRequest)
	packet, err := psbt.NewFromRawBytes(strings.NewReader(signatureRequest.Base64Psbt), true)
	if err != nil {
		return fmt.Errorf("failed to parse psbt: %w", err)
	}
	var buf bytes.Buffer
	err = packet.Serialize(&buf)
	if err != nil {
		return fmt.Errorf("failed to serialize psbt: %w", err)
	}
	finalizedPsbt := buf.Bytes()
	for i, privateKey := range c.networkConfig.PrivateKeys {
		finalize := i == len(c.networkConfig.PrivateKeys)-1
		signedPsbt, err := c.SignPsbt(finalizedPsbt, privateKey, finalize)
		if err != nil {
			return fmt.Errorf("failed to sign psbt: %w", err)
		}
		finalizedPsbt = signedPsbt
	}
	finalTx := &wire.MsgTx{}
	err = finalTx.Deserialize(bytes.NewReader(finalizedPsbt))
	if err != nil {
		return fmt.Errorf("failed to deserialize psbt: %w", err)
	}
	finalTxBuf := bytes.NewBuffer(make([]byte, 0, finalTx.SerializeSize()))
	if err := finalTx.Serialize(finalTxBuf); err != nil {
		return fmt.Errorf("failed to serialize final tx: %w", err)
	}
	if c.eventBus != nil {

		c.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_CUSTODIAL_SIGNATURES_CONFIRMED,
			DestinationChain: c.networkConfig.SignerNetwork,
			Data:             hex.EncodeToString(finalTxBuf.Bytes()),
			MessageID:        messageID,
		})
	} else {
		log.Warn().Msg("[CustodialClient] [handleBtcSignatureRequested] event bus is undefined")
	}
	return nil
}
