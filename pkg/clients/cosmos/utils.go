package cosmos

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/go-bip39"
	"github.com/rs/zerolog/log"
)

func CreateAccountFromMnemonic(mnemonic string, bip44Path string) (*secp256k1.PrivKey, types.AccAddress, error) {
	// Derive the seed from mnemonic
	seed := bip39.NewSeed(mnemonic, "")
	path := "m/44'/118'/0'/0/0"
	if bip44Path != "" {
		path = bip44Path
	}
	// Create master key and derive the private key
	// Using "m/44'/118'/0'/0/0" for Cosmos
	master, ch := hd.ComputeMastersFromSeed(seed)
	privKeyBytes, err := hd.DerivePrivateKeyForPath(master, ch, path)
	if err != nil {
		return nil, nil, err
	}

	// Create private key and get address
	privKey := &secp256k1.PrivKey{Key: privKeyBytes}
	addr := types.AccAddress(privKey.PubKey().Address())
	log.Debug().Msgf("Created account with address: %s from mnemonic: %s", addr.String(), mnemonic)
	return privKey, addr, nil
}

func ExtractCurrentSequence(err error) (int, int, error) {
	errMsg := err.Error()
	//error: code = Unknown desc = account sequence mismatch, expected 32, got 31:
	re := regexp.MustCompile(`account sequence mismatch, expected (\d+), got (\d+)`)
	match := re.FindStringSubmatch(errMsg)
	if len(match) > 2 {
		expected, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse number: %w", err)
		}
		got, err := strconv.Atoi(match[2])
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse number: %w", err)
		}
		return expected, got, nil
	} else {
		log.Error().Err(err).Msgf("[NetworkClient] [ExtractCurrentSequence] failed to extract current sequence from error message: %s", errMsg)
		return 0, 0, err
	}
}
