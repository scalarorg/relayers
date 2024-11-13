package electrs

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/scalarorg/go-electrum/electrum/types"
	"github.com/scalarorg/relayers/pkg/db/models"
	"github.com/scalarorg/relayers/pkg/utils/evm"
)

func (c *Client) CreateRelayDatas(vaultTxs []types.VaultTransaction) ([]models.RelayData, error) {
	relayDatas := make([]models.RelayData, len(vaultTxs))
	for i, vaultTx := range vaultTxs {
		relayData, err := c.CreateRelayData(vaultTx)
		if err != nil {
			return nil, err
		}
		relayDatas[i] = relayData
	}
	return relayDatas, nil
}

func (c *Client) CreateRelayData(vaultTx types.VaultTransaction) (models.RelayData, error) {
	relayData := models.RelayData{
		From: c.config.SourceChain,
		CallContract: &models.CallContract{
			TxHash:          vaultTx.TxHash,
			BlockNumber:     uint64(vaultTx.Height),
			LogIndex:        uint(vaultTx.TxPosition),
			ContractAddress: strings.ToLower(vaultTx.DestContractAddress),
			//Do not confuse this with Btc Sender address
			SourceAddress:   strings.ToLower(vaultTx.DestRecipientAddress),
			SenderAddress:   &vaultTx.StakerAddress,
			StakerPublicKey: &vaultTx.StakerPubkey,
			Amount:          vaultTx.Amount,
			Symbol:          "",
		},
	}
	chainName, err := c.commonConfig.GetChainNameById(vaultTx.DestChainId)
	if err != nil {
		return relayData, fmt.Errorf("Chain not found for input chainId: %d, %w	", vaultTx.DestChainId, err)
	}
	relayData.To = chainName
	//For btc vault tx, the log index is always 0
	relayData.ID = fmt.Sprintf("%s-%d", strings.ToLower(vaultTx.TxHash), 0)

	// Convert VaultTxHex and Payload to byte slices
	txHexBytes, err := hex.DecodeString(strings.TrimPrefix(vaultTx.TxContent, "0x"))
	if err != nil {
		return relayData, fmt.Errorf("failed to decode VaultTxHex: %w", err)
	}
	relayData.CallContract.TxHex = txHexBytes
	payloadBytes, payloadHash, err := evm.CalculatePayload(&vaultTx)
	if err != nil {
		return relayData, fmt.Errorf("failed to decode Payload: %w", err)
	}
	relayData.CallContract.Payload = payloadBytes
	relayData.CallContract.PayloadHash = strings.ToLower(payloadHash)
	return relayData, nil
}
