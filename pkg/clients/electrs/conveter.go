package electrs

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/go-electrum/electrum/types"
	relaytypes "github.com/scalarorg/relayers/pkg/types"
	"github.com/scalarorg/relayers/pkg/utils"
)

func (c *Client) CategorizeVaultTxs(vaultTxs []types.VaultTransaction) ([]*chains.TokenSent, []*relaytypes.UnstakedVaultTx) {
	tokenSents := []*chains.TokenSent{}
	unstakedVaultTxs := []*relaytypes.UnstakedVaultTx{}
	for _, vaultTx := range vaultTxs {
		if vaultTx.VaultTxType == 1 {
			//1.Staking
			tokenSent, err := c.CreateTokenSent(vaultTx)
			if err != nil {
				log.Error().Err(err).Any("BridgeTx", vaultTx).Msg("[ElectrumClient] [CreateTokenSents] failed to create token sent")
			} else if tokenSent.Symbol == "" {
				log.Error().Msgf("[ElectrumClient] [CreateTokenSents] symbol not found for token: %s", vaultTx.DestTokenAddress)
			} else {
				tokenSents = append(tokenSents, tokenSent)
			}
		} else if vaultTx.VaultTxType == 2 {
			//2.Unstaking
			unstakedVaultTx := c.CreateUnstakedVaultTx(vaultTx)
			unstakedVaultTxs = append(unstakedVaultTxs, unstakedVaultTx)
			log.Info().Any("RedeemTx", unstakedVaultTx).Msg("[ElectrumClient] [CategorizeVaultTxs]")
		}
	}
	return tokenSents, unstakedVaultTxs
}

func (c *Client) CreateTokenSent(vaultTx types.VaultTransaction) (*chains.TokenSent, error) {
	//For btc vault tx, the log index is tx position in the block
	index := vaultTx.TxPosition
	eventId := fmt.Sprintf("%s-%d", utils.NormalizeHash(vaultTx.TxHash), index)

	chainInfo, err := utils.ConvertUint64ToChainInfo(vaultTx.DestChain)
	if err != nil {
		return nil, fmt.Errorf("failed to convert uint64 to chain info: %w", err)
	}
	//parse chain id to chain name

	destinationChainName, err := c.globalConfig.GetStringIdByChainId(chainInfo.ChainType.String(), chainInfo.ChainID)
	if err != nil {
		return nil, fmt.Errorf("chain not found for input chainId: %v, %w	", chainInfo, err)
	}
	symbol, err := c.GetSymbol(destinationChainName, vaultTx.DestTokenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get symbol: %w", err)
	}
	destAddress := utils.NormalizeAddress(vaultTx.DestRecipientAddress, chainInfo.ChainType)

	tokenSent := chains.TokenSent{
		EventID:              eventId,
		TxHash:               vaultTx.TxHash,
		BlockNumber:          uint64(vaultTx.Height),
		LogIndex:             uint(vaultTx.TxPosition),
		SourceChain:          c.electrumConfig.SourceChain,
		SourceAddress:        strings.ToLower(vaultTx.StakerAddress),
		DestinationChain:     destinationChainName,
		DestinationAddress:   destAddress,
		Symbol:               symbol,
		TokenContractAddress: vaultTx.DestTokenAddress,
		Amount:               vaultTx.Amount,
		Status:               chains.TokenSentStatusPending,
		CreatedAt:            time.Unix(int64(vaultTx.Timestamp), 0),
		UpdatedAt:            time.Unix(int64(vaultTx.Timestamp), 0),
	}
	return &tokenSent, nil
}

func (c *Client) CreateUnstakedVaultTx(vaultTx types.VaultTransaction) *relaytypes.UnstakedVaultTx {
	return &relaytypes.UnstakedVaultTx{
		BlockHeight: uint64(vaultTx.Height),
		TxHash:      vaultTx.TxHash,
		LogIndex:    uint(vaultTx.TxPosition),
	}
}
