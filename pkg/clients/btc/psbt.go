package btc

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	vault "github.com/scalarorg/bitcoin-vault/ffi/go-vault"
	utils "github.com/scalarorg/bitcoin-vault/go-utils/types"
	"github.com/scalarorg/relayers/pkg/types"
	covtypes "github.com/scalarorg/scalar-core/x/covenant/types"
)

/*
* Create psbt based on taproot address and query command response as utxo output
* Todo: Seperate psbt by params: feeOpts, rbf
 */
func (c *BtcClient) createPsbts(psbtParams types.PsbtParams, outpoints []CommandOutPoint) ([]covtypes.Psbt, error) {
	//Todo: handle custom replace by fee
	replaceByFee := false
	psbts := []covtypes.Psbt{}
	taprootAddress, err := psbtParams.GetTaprootAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get taproot address: %w", err)
	}
	utxos, err := c.GetAddressTxsUtxo(taprootAddress.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get utxo list: %w", err)
	}
	//Group output by fee option
	mapOutpoints := map[uint64][]vault.UnstakingOutput{}
	for _, outpoint := range outpoints {
		//TODO: handle fee opts
		feeOpts := uint64(outpoint.BTCFeeOpts)
		feeOpts = uint64(utils.MinimumFee)
		outpoints, ok := mapOutpoints[feeOpts]
		if !ok {
			outpoints = []vault.UnstakingOutput{}
		}
		mapOutpoints[feeOpts] = append(outpoints, outpoint.OutPoint)
	}
	// space out utxos by fee opts
	mapUtxos := map[uint64][]vault.PreviousStakingUTXO{}
	prevUtxos := make([]vault.PreviousStakingUTXO, 0, len(utxos))
	for _, utxo := range utxos {
		txid, err := chainhash.NewHashFromStr(utxo.Txid)
		if err != nil {
			return nil, fmt.Errorf("invalid txid %s: %w", utxo.Txid, err)
		}
		prevUtxos = append(prevUtxos, vault.PreviousStakingUTXO{
			OutPoint: vault.OutPoint{
				Txid: [32]byte(txid.CloneBytes()),
				Vout: utxo.Vout,
			},
			Amount: utxo.Value,
			Script: psbtParams.CovenantScript,
		})
	}
	mapUtxos[uint64(utils.MinimumFee)] = prevUtxos
	for feeOpts, outpoints := range mapOutpoints {
		psbt, err := vault.BuildCovenantOnlyUnstakingTx(
			psbtParams.ScalarTag,
			psbtParams.ProtocolTag,
			psbtParams.Version,
			psbtParams.NetworkKind,
			mapUtxos[feeOpts],
			outpoints,
			psbtParams.CovenantPubKey,
			psbtParams.CovenantQuorum,
			replaceByFee,
			uint64(feeOpts),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to build psbt: %w", err)
		}
		psbts = append(psbts, psbt)
	}
	return psbts, nil
}
