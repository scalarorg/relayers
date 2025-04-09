package utils

import (
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/bitcoin-vault/go-utils/encode"
)

func DecodeContractCallWithTokenPayload(payload []byte) (*encode.ContractCallWithTokenPayload, error) {
	// the DecodeContractCallWithTokenPayload also detect the payload type, the reason we decode again when failed is to ensure compatibility with old payloads
	decodedPayload, err := encode.DecodeContractCallWithTokenPayload(payload)
	if err == nil {
		log.Info().Any("DecodeContractCallWithTokenPayload", decodedPayload).Msg("decodedPayload")
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
