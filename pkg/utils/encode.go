package utils

import "github.com/scalarorg/bitcoin-vault/go-utils/encode"

func DecodeContractCallWithTokenPayload(payload []byte) (*encode.ContractCallWithTokenPayload, error) {
	decodedPayload, err := encode.DecodeContractCallWithTokenPayload(payload)
	if err == nil {
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
