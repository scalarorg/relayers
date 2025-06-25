package electrs

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	dataTypes "github.com/scalarorg/data-models/types"
	"github.com/scalarorg/go-electrum/electrum/types"
	"github.com/scalarorg/relayers/pkg/utils"
	"gorm.io/gorm"
)

func ReverseProofs(proofs []string) []string {
	reversedProofs := make([]string, len(proofs))
	for i, proof := range proofs {
		txHash, err := chainhash.NewHashFromStr(proof)
		if err != nil {
			log.Error().Err(err).Msgf("[ElectrumClient] [ReverseProofs] failed to get tx hash for proof: %s", proof)
			continue
		}
		reversedProofs[i] = hex.EncodeToString(txHash.CloneBytes())
	}
	return reversedProofs
}

func (c *Client) CategorizeVaultBlock(vaultBlock *types.VaultBlock) ([]*chains.TokenSent, []*chains.BtcRedeemTx) {
	tokenSents := []*chains.TokenSent{}
	redeemTxs := []*chains.BtcRedeemTx{}
	//Handle reorg, process only tx with highest block number
	log.Debug().Str("BlockHash", vaultBlock.Hash).
		Int("BlockHeight", vaultBlock.Height).
		Msg("[ElectrumClient] [CategorizeVaultTxs] process vault blocks")
	for _, vaultTx := range vaultBlock.VaultTxs {
		txInfo := vaultTx.TxInfo
		if txInfo.VaultTxType == 1 {
			//1.Staking
			tokenSent, err := c.CreateTokenSent(txInfo)
			if err != nil {
				log.Error().Err(err).Msg("[ElectrumClient] [CreateTokenSents] failed to create token sent")
			} else if tokenSent != nil {
				if tokenSent.Symbol == "" {
					log.Error().Msgf("[ElectrumClient] [CreateTokenSents] symbol not found for token: %s", txInfo.DestTokenAddress)
				} else {
					tokenSent.RawTx = vaultTx.RawTx
					tokenSent.MerkleProof = dataTypes.StringArray(ReverseProofs(vaultTx.Proof))
					tokenSent.BlockNumber = uint64(vaultBlock.Height)
					tokenSent.BlockTime = uint64(vaultBlock.Time)
					tokenSents = append(tokenSents, tokenSent)
				}
			}
		} else if txInfo.VaultTxType == 2 {
			//2.Unstaking
			redeemTx, err := c.CreateRedeemTx(txInfo)
			if err != nil {
				log.Error().Err(err).Msg("[ElectrumClient] failed to create redeem tx")
			} else {
				redeemTx.RawTx = vaultTx.RawTx
				redeemTx.MerkleProof = dataTypes.StringArray(ReverseProofs(vaultTx.Proof))
				redeemTx.BlockTime = uint64(vaultBlock.Time)
				redeemTxs = append(redeemTxs, redeemTx)
				log.Info().Any("RedeemTx", redeemTx).Msg("[ElectrumClient] [CategorizeVaultTxs]")
			}
		}
	}
	return tokenSents, redeemTxs
}

func (c *Client) CategorizeVaultTxs(vaultTxs []types.VaultTransaction) ([]*chains.TokenSent, []*chains.BtcRedeemTx) {
	tokenSents := []*chains.TokenSent{}
	redeemTxs := []*chains.BtcRedeemTx{}
	//Handle reorg, process only tx with highest block number
	mapTxBlockNumbers := make(map[string]int)
	for _, vaultTx := range vaultTxs {
		blockNumber := mapTxBlockNumbers[vaultTx.TxHash]
		if blockNumber > vaultTx.Height {
			log.Debug().Str("VaultTxHash", vaultTx.TxHash).
				Int("BlockNumber", vaultTx.Height).
				Int("ExistingBlockNumber", blockNumber).
				Msg("[ElectrumClient] [CategorizeVaultTxs] skip tx")
			continue
		} else {
			mapTxBlockNumbers[vaultTx.TxHash] = vaultTx.Height
			if blockNumber == 0 {
				log.Debug().Str("VaultTxHash", vaultTx.TxHash).
					Int("BlockNumber", vaultTx.Height).
					Msg("[ElectrumClient] [CategorizeVaultTxs] process tx")
			} else {
				log.Debug().Str("VaultTxHash", vaultTx.TxHash).
					Int("BlockNumber", vaultTx.Height).
					Int("ExistingBlockNumber", blockNumber).
					Msg("[ElectrumClient] [CategorizeVaultTxs] process tx")
			}
		}
		if vaultTx.VaultTxType == 1 {
			//1.Staking
			tokenSent, err := c.CreateTokenSent(vaultTx)
			if err != nil {
				log.Error().Err(err).Msg("[ElectrumClient] [CreateTokenSents] failed to create token sent")
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
	stakerPubkey, err := hex.DecodeString(vaultTx.StakerPubkey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode staker pubkey: %w", err)
	}
	tokenSent := chains.TokenSent{
		EventID:              eventId,
		TxHash:               vaultTx.TxHash,
		TxPosition:           uint64(vaultTx.TxPosition),
		BlockNumber:          uint64(vaultTx.Height),
		BlockTime:            uint64(vaultTx.Timestamp),
		LogIndex:             uint(vaultTx.TxPosition),
		SourceChain:          c.electrumConfig.SourceChain,
		SourceAddress:        strings.ToLower(vaultTx.StakerAddress),
		StakerPubkey:         stakerPubkey,
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

func (c *Client) CreateRedeemTx(vaultTx types.VaultTransaction) (*chains.BtcRedeemTx, error) {
	//We redeem by custodian group which can be shared between protocol,
	//so we can't store tokenaddress and symbol here
	groupUid := hex.EncodeToString(vaultTx.CustodianGroupUid)
	for _, group := range c.custodialGroups {
		if bytes.Equal(vaultTx.CustodianGroupUid, group.UID[:]) {
			return &chains.BtcRedeemTx{
				Model: gorm.Model{
					CreatedAt: time.Unix(int64(vaultTx.Timestamp), 0),
					UpdatedAt: time.Unix(int64(vaultTx.Timestamp), 0),
				},
				Chain:             c.electrumConfig.SourceChain,
				BlockNumber:       uint64(vaultTx.Height),
				BlockTime:         uint64(vaultTx.Timestamp),
				TxHash:            vaultTx.TxHash,
				TxPosition:        uint64(vaultTx.TxPosition),
				Amount:            vaultTx.Amount,
				SessionSequence:   vaultTx.SessionSequence,
				CustodianGroupUid: groupUid,
				Status:            string(chains.RedeemStatusExecuting),
			}, nil
		}
	}
	return nil, fmt.Errorf("custodian group uid is not valid %s", groupUid)
}
