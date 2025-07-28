package utils

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/go-common/encode"
)

func DecodeContractCallWithTokenPayload(payload []byte) (*encode.ContractCallWithTokenPayload, error) {
	// the DecodeContractCallWithTokenPayload also detect the payload type, the reason we decode again when failed is to ensure compatibility with old payloads
	decodedPayload, err := encode.DecodeContractCallWithTokenPayload(payload)
	if err == nil {
		if decodedPayload.CustodianOnly != nil {
			log.Info().Any("DecodeContractCallWithTokenPayload", decodedPayload).
				Str("Redeem Address", hex.EncodeToString(decodedPayload.CustodianOnly.RecipientChainIdentifier)).
				Msg("decodedPayload")
		} else {
			log.Info().Any("DecodeContractCallWithTokenPayload", decodedPayload).
				Msg("decodedPayload")
		}
		return decodedPayload, nil
	}
	decodedPayload, err = encode.DecodeCustodianOnly(payload)
	if err == nil {
		return decodedPayload, nil
	}
	decodedPayload, err = encode.DecodeUPC(payload)
	if err == nil {
		return decodedPayload, nil
	}
	return nil, err
}

func DecodeGroupUid(groupHex string) ([32]byte, error) {
	groupBytes, err := hex.DecodeString(strings.TrimPrefix(groupHex, "0x"))
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to decode group uid: %w", err)
	}
	groupBytes32 := [32]byte{}
	copy(groupBytes32[:], groupBytes)
	return groupBytes32, nil
}
