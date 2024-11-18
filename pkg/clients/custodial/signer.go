package custodial

import (
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	psbtFfi "github.com/scalarorg/bitcoin-vault/ffi/go-psbt"
)

func (c *Client) SignPsbt(inputBytes []byte, privateKey string, finalize bool) ([]byte, error) {
	wif, err := btcutil.DecodeWIF(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode WIF: %w", err)
	}
	privKeyBytes := wif.PrivKey.Serialize()
	var networkKind psbtFfi.NetworkKind
	if c.networkConfig.SignerNetwork == "testnet" {
		networkKind = psbtFfi.NetworkKindTestnet
	} else {
		networkKind = psbtFfi.NetworkKindMainnet
	}
	partialSignedPsbt, err := psbtFfi.SignPsbtBySingleKey(
		inputBytes,      // []byte containing PSBT
		privKeyBytes[:], // []byte containing private key
		networkKind,     // bool indicating if testnet
		finalize,        // finalize
	)
	return partialSignedPsbt, nil
}
