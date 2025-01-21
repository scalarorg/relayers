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
func (c *BtcClient) CreatePsbts(psbtParams types.PsbtParams, outpoints []CommandOutPoint) ([]covtypes.Psbt, error) {
	//Todo: handle custom replace by fee
	replaceByFee := false
	psbts := []covtypes.Psbt{}

	taprootAddress, err := psbtParams.GetTaprootAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get taproot address: %w", err)
	}
	totalAmount := uint64(0)
	for _, outpoint := range outpoints {
		totalAmount += outpoint.OutPoint.Amount
	}
	utxos, err := c.GetAddressTxsUtxo(taprootAddress.String(), totalAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to get utxo list: %w", err)
	}

	//Group output by fee option
	mapOutpoints := map[uint64][]utils.UnstakingOutput{}
	for _, outpoint := range outpoints {
		//TODO: handle fee opts
		// feeOpts := uint64(outpoint.BTCFeeOpts)
		feeOpts := uint64(utils.MinimumFee)
		outpoints, ok := mapOutpoints[feeOpts]
		if !ok {
			outpoints = []utils.UnstakingOutput{}
		}
		mapOutpoints[feeOpts] = append(outpoints, outpoint.OutPoint)
	}
	// space out utxos by fee opts
	mapUtxos := map[uint64][]utils.PreviousStakingUTXO{}
	prevUtxos := make([]utils.PreviousStakingUTXO, 0, len(utxos))
	for _, utxo := range utxos {
		txid, err := chainhash.NewHashFromStr(utxo.Txid)
		if err != nil {
			return nil, fmt.Errorf("invalid txid %s: %w", utxo.Txid, err)
		}
		prevUtxos = append(prevUtxos, utils.PreviousStakingUTXO{
			OutPoint: utils.OutPoint{
				Txid: [32]byte(txid.CloneBytes()),
				Vout: utxo.Vout,
			},
			Amount: utxo.Value,
			Script: psbtParams.CustodianScript,
		})
	}
	mapUtxos[uint64(utils.MinimumFee)] = prevUtxos
	for feeOpts, outpoints := range mapOutpoints {
		psbt, err := vault.BuildCustodianOnlyUnstakingTx(
			psbtParams.ScalarTag,
			psbtParams.ProtocolTag,
			psbtParams.Version,
			psbtParams.NetworkKind,
			mapUtxos[feeOpts],
			outpoints,
			psbtParams.CustodianPubKey,
			psbtParams.CustodianQuorum,
			replaceByFee,
			uint64(1), // TODO: handle fee opts
		)

		if err != nil {
			return nil, fmt.Errorf("failed to build psbt: %w", err)
		}
		psbts = append(psbts, psbt)
	}
	return psbts, nil
}
