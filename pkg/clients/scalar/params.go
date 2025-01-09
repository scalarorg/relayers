package scalar

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/bitcoin-vault/ffi/go-vault"
	"github.com/scalarorg/relayers/pkg/types"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	protypes "github.com/scalarorg/scalar-core/x/protocol/types"
	scalarnettypes "github.com/scalarorg/scalar-core/x/scalarnet/types"
)

func (c *Client) GetPsbtParams(chainName string) types.PsbtParams {
	var params types.PsbtParams
	// 1. Get scalarnet params
	scalarnetClient, err := c.GetQueryClient().GetScalarnetQueryClient()
	if err != nil {
		log.Error().Err(err).Msgf("[ScalarClient] [GetPsbtParams] cannot create query client")
	}
	scalarnetRes, err := scalarnetClient.Params(context.Background(), &scalarnettypes.ParamsRequest{})
	if err != nil {
		log.Error().Msgf("[ScalarClient] [GetPsbtParams] error: %v", err)
	}
	params.ScalarTag = scalarnetRes.Params.Tag
	params.Version = uint8(scalarnetRes.Params.Version)
	//2. Get chain params
	chainClient, err := c.GetQueryClient().GetChainQueryServiceClient()
	if err != nil {
		log.Error().Err(err).Msgf("[ScalarClient] [GetPsbtParams] cannot create query client")
	}
	chainRes, err := chainClient.Params(context.Background(), &chainstypes.ParamsRequest{
		Chain: chainName,
	})
	if err != nil {
		log.Error().Msgf("[ScalarClient] [GetPsbtParams] error: %v", err)
	}
	params.NetworkKind = vault.NetworkKind(chainRes.Params.NetworkKind)
	params.NetworkType = chainRes.Params.Metadata["params"]
	//3. Get protocol params
	protocolClient, err := c.GetQueryClient().GetProtocolQueryClient()
	if err != nil {
		log.Error().Err(err).Msgf("[ScalarClient] [GetPsbtParams] cannot create protocol query client")
	}
	protocols, err := protocolClient.Protocols(context.Background(), &protypes.ProtocolsRequest{})
	if err != nil {
		log.Error().Err(err).Msgf("[ScalarClient] [GetPsbtParams] cannot get protocols")
	}
	if len(protocols.Protocols) == 0 {
		log.Error().Msgf("[ScalarClient] [GetPsbtParams] no protocols found")
	} else {
		//Get first protocol
		protocol := protocols.Protocols[0]
		params.CovenantPubKey = make([]vault.PublicKey, len(protocol.CustodianGroup.Custodians))
		for i, custodian := range protocol.CustodianGroup.Custodians {
			params.CovenantPubKey[i] = vault.PublicKey(custodian.BtcPubkey)
		}
		params.CovenantQuorum = uint8(protocol.CustodianGroup.Quorum)
		params.CovenantScript = protocol.CustodianGroup.BtcPubkey
		params.ProtocolTag = protocol.Tag
	}
	return params
}