package btc

import (
	"fmt"

	"github.com/btcsuite/btcd/wire"
	covtypes "github.com/scalarorg/scalar-core/x/covenant/types"
)

/*
* Create psbt based on taproot address and query command response as utxo output
 */
func (c *BtcClient) createPsbt(taprootAddress string, outpoints []*wire.TxOut) (covtypes.Psbt, error) {
	_, err := c.GetAddressTxsUtxo(taprootAddress)
	if err != nil {
		return covtypes.Psbt{}, fmt.Errorf("failed to get utxo list: %w", err)
	}
	//TODO: create psbt
	return covtypes.Psbt{}, nil
}
