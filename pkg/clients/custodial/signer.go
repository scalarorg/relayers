package custodial

import (
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	vault "github.com/scalarorg/bitcoin-vault/ffi/go"
	"github.com/scalarorg/go-common/types"
)

func (c *Client) SignPsbt(inputBytes []byte, privateKey string, finalize bool) ([]byte, error) {
	wif, err := btcutil.DecodeWIF(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode WIF: %w", err)
	}
	privKeyBytes := wif.PrivKey.Serialize()
	var networkKind types.NetworkKind
	// TODO: Cross-check with xchains core's btc module
	if c.networkConfig.SignerNetwork == "bitcoin" {
		networkKind = types.NetworkKindMainnet
	} else {
		networkKind = types.NetworkKindTestnet
	}
	partialSignedPsbt, err := vault.SignPsbtBySingleKey(
		inputBytes,      // []byte containing PSBT
		privKeyBytes[:], // []byte containing private key
		networkKind,     // bool indicating if testnet
		finalize,        // finalize
	)
	return partialSignedPsbt, nil
}
