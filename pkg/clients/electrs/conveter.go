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
	relaytypes "github.com/scalarorg/relayers/pkg/types"
)

func (c *Client) CategorizeVaultTxs(vaultTxs []types.VaultTransaction) ([]*chains.TokenSent, []*relaytypes.UnstakedVaultTx) {
	tokenSents := []*chains.TokenSent{}
	unstakedVaultTxs := []*relaytypes.UnstakedVaultTx{}
	for _, vaultTx := range vaultTxs {
		if vaultTx.VaultTxType == 1 {
			//1.Staking
			tokenSent, err := c.CreateTokenSent(vaultTx)
			if err != nil {
				log.Error().Err(err).Any("VaultTx", vaultTx).Msgf("[ElectrumClient] [CreateTokenSents] failed to create token sent: %v", vaultTx)
			} else if tokenSent.Symbol == "" {
				log.Error().Msgf("[ElectrumClient] [CreateTokenSents] symbol not found for token: %s", vaultTx.DestTokenAddress)
			} else {
				tokenSents = append(tokenSents, &tokenSent)
			}
		} else if vaultTx.VaultTxType == 2 {
			//2.Unstaking
			unstakedVaultTx := c.CreateUnstakedVaultTx(vaultTx)
			unstakedVaultTxs = append(unstakedVaultTxs, unstakedVaultTx)
		}
	}
	return tokenSents, unstakedVaultTxs
}

func (c *Client) CreateTokenSent(vaultTx types.VaultTransaction) (chains.TokenSent, error) {
	//For btc vault tx, the log index is tx position in the block
	index := vaultTx.TxPosition
	eventId := fmt.Sprintf("%s-%d", strings.ToLower(vaultTx.TxHash), index)
	eventId = strings.TrimPrefix(eventId, "0x")
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
	symbol, err := c.GetSymbol(destinationChainName, vaultTx.DestTokenAddress)
	if err != nil {
		return tokenSent, fmt.Errorf("failed to get symbol: %w", err)
	}
	tokenSent.Symbol = symbol
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

func (c *Client) CreateUnstakedVaultTx(vaultTx types.VaultTransaction) *relaytypes.UnstakedVaultTx {
	return &relaytypes.UnstakedVaultTx{}
}
