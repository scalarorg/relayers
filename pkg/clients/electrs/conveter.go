package electrs

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/scalarorg/bitcoin-vault/go-utils/chain"
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
	//For btc vault tx, the log index is tx position in the block
	id := fmt.Sprintf("%s-%d", strings.ToLower(vaultTx.TxHash), vaultTx.TxPosition)
	relayData := models.RelayData{
		ID:   id,
		From: c.electrumConfig.SourceChain,
		CallContract: &models.CallContract{
			ID:              id,
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
	//parse chain id to chain name
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, vaultTx.DestChain)
	destinationChain := chain.NewDestinationChainFromBytes(buf)
	if destinationChain == nil {
		return relayData, fmt.Errorf("invalid destination chain: %d", vaultTx.DestChain)
	}
	destinationChainName, err := c.globalConfig.GetStringIdByChainId(destinationChain.ChainType.String(), destinationChain.ChainID)
	if err != nil {
		return relayData, fmt.Errorf("chain not found for input chainId: %v, %w	", destinationChain, err)
	}
	relayData.To = destinationChainName

	// Convert VaultTxHex and Payload to byte slices
	txHexBytes, err := hex.DecodeString(strings.TrimPrefix(vaultTx.TxContent, "0x"))
	if err != nil {
		return relayData, fmt.Errorf("failed to decode VaultTxHex: %w", err)
	}
	relayData.CallContract.TxHex = txHexBytes
	payloadBytes, payloadHash, err := evm.CalculateStakingPayload(&vaultTx)
	if err != nil {
		return relayData, fmt.Errorf("failed to decode Payload: %w", err)
	}
	relayData.CallContract.Payload = payloadBytes
	relayData.CallContract.PayloadHash = strings.ToLower(payloadHash)
	return relayData, nil
}
