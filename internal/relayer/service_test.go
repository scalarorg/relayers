package relayer_test

import (
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/internal/relayer"
	"github.com/scalarorg/relayers/pkg/clients/evm"
	chainExported "github.com/scalarorg/scalar-core/x/chains/exported"
	covExported "github.com/scalarorg/scalar-core/x/covenant/exported"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

var (
	mockCustodianGroupUid = sha3.Sum256([]byte("scalarv36"))
	mockCustodianGroup    = &covExported.CustodianGroup{
		UID: chainExported.Hash(mockCustodianGroupUid[:]),
	}
	globalConfig  config.Config         = config.Config{}
	sepoliaConfig *evm.EvmNetworkConfig = &evm.EvmNetworkConfig{
		ChainID:      11155111,
		ID:           "evm|11155111",
		Name:         "Ethereum sepolia",
		RPCUrl:       "wss://eth-sepolia.g.alchemy.com/v2/nNbspp-yjKP9GtAcdKi8xcLnBTptR2Zx",
		AuthWeighted: "0xE3ff6841C7B9a9b540eD7207b2B88D65A138FB4a",
		Gateway:      "0xCd60852A48fc101304C603A9b1Bbd1E40d35E8c8", //Version Apr 15, 2025
		PrivateKey:   "",
		Finality:     1,
		BlockTime:    time.Second * 12,
		StartBlock:   8095740,
		RecoverRange: 1000000,
		GasLimit:     300000,
	}
	bnbConfig *evm.EvmNetworkConfig = &evm.EvmNetworkConfig{
		ChainID:      97,
		ID:           "evm|97",
		Name:         "Ethereum bnb",
		RPCUrl:       "wss://bnb-testnet.g.alchemy.com/v2/DpCscOiv_evEPscGYARI3cOVeJ59CRo8",
		AuthWeighted: "0x13bB2b1240E582C1A3519E97a08157eD1bBD36Bf",
		Gateway:      "0x930C3c4f7d26f18830318115DaD97E0179DA55f0",
		PrivateKey:   "",
		Finality:     1,
		BlockTime:    time.Second * 12,
		StartBlock:   47254017,
		GasLimit:     300000,
	}
)

// CGO_LDFLAGS="-L./lib -lbitcoin_vault_ffi" CGO_CFLAGS="-I./lib" go test -timeout 10m -run ^TestRecoverEvmSessions$ github.com/scalarorg/relayers/internal/relayer -v -count=1
func TestRecoverEvmSessions(t *testing.T) {
	service := createMockService()
	groups := []chainExported.Hash{
		(chainExported.Hash)(mockCustodianGroupUid),
	}
	err := service.RecoverEvmSessions(groups[0])
	require.NoError(t, err)
}

func createMockService() *relayer.Service {
	sepoliaClient, err := evm.NewEvmClient(&globalConfig, sepoliaConfig, nil, nil, nil)
	if err != nil {
		log.Error().Msgf("failed to create evm client: %v", err)
	}
	bnbClient, err := evm.NewEvmClient(&globalConfig, bnbConfig, nil, nil, nil)
	if err != nil {
		log.Error().Msgf("failed to create evm client: %v", err)
	}
	service := &relayer.Service{
		EvmClients: []*evm.EvmClient{
			sepoliaClient,
			bnbClient,
		},
	}
	return service
}
