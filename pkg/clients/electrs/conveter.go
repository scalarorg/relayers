package electrs

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/go-electrum/electrum/types"
	"github.com/scalarorg/relayers/pkg/utils"
	"gorm.io/gorm"
)

func (c *Client) CategorizeVaultTxs(vaultTxs []types.VaultTransaction) ([]*chains.TokenSent, []*chains.RedeemTx) {
	tokenSents := []*chains.TokenSent{}
	redeemTxs := []*chains.RedeemTx{}
	for _, vaultTx := range vaultTxs {
		if vaultTx.VaultTxType == 1 {
			//1.Staking
			tokenSent, err := c.CreateTokenSent(vaultTx)
			if err != nil {
				log.Error().Err(err).Any("BridgeTx", vaultTx).Msg("[ElectrumClient] [CreateTokenSents] failed to create token sent")
			} else if tokenSent != nil {
				if tokenSent.Symbol == "" {
					log.Error().Msgf("[ElectrumClient] [CreateTokenSents] symbol not found for token: %s", vaultTx.DestTokenAddress)
				} else {
					tokenSents = append(tokenSents, tokenSent)
				}
			}
		} else if vaultTx.VaultTxType == 2 {
			//2.Unstaking
			redeemTx, err := c.CreateRedeemTx(vaultTx)
			if err != nil {
				log.Error().Err(err).Msg("[ElectrumClient] failed to create redeem tx")
			} else {
				redeemTxs = append(redeemTxs, redeemTx)
				log.Info().Any("RedeemTx", redeemTx).Msg("[ElectrumClient] [CategorizeVaultTxs]")
			}
		}
	}
	return tokenSents, redeemTxs
}

func (c *Client) CreateTokenSent(vaultTx types.VaultTransaction) (*chains.TokenSent, error) {
	//Check if vaultTx.scriptPubkey is btcPubkey of a custodianGroup
	foundCustodianGroup := false
	for _, group := range c.custodialGroups {
		if bytes.Equal(vaultTx.ScriptPubkey, group.BitcoinPubkey) {
			foundCustodianGroup = true
			break
		}
	}
	if !foundCustodianGroup {
		pubkeys := []string{}
		for _, group := range c.custodialGroups {
			pubkeys = append(pubkeys, hex.EncodeToString(group.BitcoinPubkey))
		}
		log.Debug().Strs("Supported custodian groups taproot pubkey", pubkeys).Msg("[ElectrumClient] CreateTokenSent failed")
		return nil, fmt.Errorf("custodian group for script pubkey %s not found", hex.EncodeToString(vaultTx.ScriptPubkey))
	}
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
		BlockTime:            uint64(vaultTx.Timestamp),
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

func (c *Client) CreateRedeemTx(vaultTx types.VaultTransaction) (*chains.RedeemTx, error) {
	//We redeem by custodian group which can be shared between protocol,
	//so we can't store tokenaddress and symbol here
	groupUid := hex.EncodeToString(vaultTx.CustodianGroupUid)
	for _, group := range c.custodialGroups {
		if bytes.Equal(vaultTx.CustodianGroupUid, group.UID[:]) {
			return &chains.RedeemTx{
				Model: gorm.Model{
					CreatedAt: time.Unix(int64(vaultTx.Timestamp), 0),
					UpdatedAt: time.Unix(int64(vaultTx.Timestamp), 0),
				},
				Chain:             c.electrumConfig.SourceChain,
				BlockNumber:       uint64(vaultTx.Height),
				BlockTime:         uint64(vaultTx.Timestamp),
				TxHash:            vaultTx.TxHash,
				Amount:            vaultTx.Amount,
				SessionSequence:   vaultTx.SessionSequence,
				CustodianGroupUid: groupUid,
				Status:            string(chains.RedeemStatusExecuting),
			}, nil
		}
	}
	return nil, fmt.Errorf("custodian group uid is not valid %s", groupUid)
}
