package electrs

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/bitcoin-vault/go-utils/chain"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/go-electrum/electrum/types"
)

func (c *Client) CreateTokenSents(vaultTxs []types.VaultTransaction) ([]*chains.TokenSent, error) {
	tokenSents := []*chains.TokenSent{}
	for _, vaultTx := range vaultTxs {
		tokenSent, err := c.CreateTokenSent(vaultTx)
		if err != nil {
			return nil, err
		}
		if tokenSent.Symbol == "" {
			log.Error().Msgf("[ElectrumClient] [CreateTokenSents] symbol not found for token: %s", vaultTx.DestTokenAddress)
		} else {
			tokenSents = append(tokenSents, &tokenSent)
		}
	}
	return tokenSents, nil
}

func (c *Client) CreateTokenSent(vaultTx types.VaultTransaction) (chains.TokenSent, error) {
	//For btc vault tx, the log index is tx position in the block
	index := vaultTx.TxPosition
	eventId := fmt.Sprintf("%s-%d", strings.ToLower(vaultTx.TxHash), index)
	tokenSent := chains.TokenSent{
		EventID:              eventId,
		TxHash:               vaultTx.TxHash,
		BlockNumber:          uint64(vaultTx.Height),
		LogIndex:             uint(vaultTx.TxPosition),
		SourceChain:          c.electrumConfig.SourceChain,
		SourceAddress:        strings.ToLower(vaultTx.StakerAddress),
		DestinationAddress:   strings.ToLower(vaultTx.DestRecipientAddress),
		TokenContractAddress: vaultTx.DestTokenAddress,
		Amount:               vaultTx.Amount,
		Status:               chains.TokenSentStatusPending,
		CreatedAt:            time.Unix(int64(vaultTx.Timestamp), 0),
		UpdatedAt:            time.Unix(int64(vaultTx.Timestamp), 0),
	}
	//parse chain id to chain name
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, vaultTx.DestChain)
	chainInfo := chain.NewChainInfoFromBytes(buf)
	if chainInfo == nil {
		return tokenSent, fmt.Errorf("invalid destination chain: %d", vaultTx.DestChain)
	}
	destinationChainName, err := c.globalConfig.GetStringIdByChainId(chainInfo.ChainType.String(), chainInfo.ChainID)
	if err != nil {
		return tokenSent, fmt.Errorf("chain not found for input chainId: %v, %w	", chainInfo, err)
	}
	tokenSent.DestinationChain = destinationChainName
	tokenSent.Symbol = c.GetSymbol(destinationChainName, vaultTx.DestTokenAddress)
	// Convert VaultTxHex and Payload to byte slices
	// txHexBytes, err := hex.DecodeString(strings.TrimPrefix(vaultTx.TxContent, "0x"))
	// if err != nil {
	// 	return relayData, fmt.Errorf("failed to decode VaultTxHex: %w", err)
	// }
	// relayData.CallContract.TxHex = txHexBytes
	// payloadBytes, payloadHash, err := evm.CalculateStakingPayload(&vaultTx)
	// if err != nil {
	// 	return relayData, fmt.Errorf("failed to decode Payload: %w", err)
	// }
	// relayData.CallContract.Payload = payloadBytes
	// relayData.CallContract.PayloadHash = strings.ToLower(payloadHash)
	return tokenSent, nil
}
