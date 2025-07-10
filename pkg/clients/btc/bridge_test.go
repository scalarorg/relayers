package btc_test

import (
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/rs/zerolog/log"
	chainsExported "github.com/scalarorg/scalar-core/x/chains/exported"
)

func TestHashFromHex(t *testing.T) {
	btcTxHash := "dc36928e2bbfaef6b1ad8b6e943490b0eb94393b7f85d69e15b39bb7078e9f80"
	chainHash, err := chainhash.NewHashFromStr(btcTxHash)
	if err != nil {
		t.Fatal(err)
	}
	log.Info().Str("chainHash", chainHash.String()).Msg("chainHash")
	evmHash, err := chainsExported.HashFromHex(btcTxHash)
	if err != nil {
		t.Fatal(err)
	}
	log.Info().Str("evmHash", evmHash.String()).Msg("evmHash")
	fmt.Println(evmHash.String())

	txHash, err := chainsExported.HashFromHex(btcTxHash)
	if err != nil {
		t.Fatal(err)
	}
	log.Info().Str("txHash", txHash.String()).Msg("txHash")
	fmt.Println(txHash.String())
}
