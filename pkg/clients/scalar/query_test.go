package scalar_test

import (
	"context"
	"testing"

	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
)

func TestGetCovenantGroups(t *testing.T) {
	scalarClient, err := scalar.NewClientFromConfig(&config.GlobalConfig, &ScalarNetworkConfig, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create scalar client: %v", err)
	}
	covenantGroups, err := scalarClient.GetCovenantGroups(context.Background())
	if err != nil {
		t.Fatalf("Failed to get covenant groups: %v", err)
	}
	for _, covenantGroup := range covenantGroups {
		t.Logf("Covenant group: %v", covenantGroup)
	}
}

func TestGetChainRedeemSession(t *testing.T) {
	scalarClient, err := scalar.NewClientFromConfig(&config.GlobalConfig, &ScalarNetworkConfig, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create scalar client: %v", err)
	}
	custodianGroupUid := [32]byte{62, 121, 50, 106, 148, 147, 137, 110, 19, 175, 98, 25, 78, 105, 77, 255, 76, 147, 0, 112, 4, 7, 68, 147, 99, 86, 75, 14, 174, 175, 7, 232}
	redeemSession, err := scalarClient.GetChainRedeemSession("evm|11155111", custodianGroupUid)
	if err != nil {
		t.Fatalf("Failed to get redeem session: %v", err)
	}
	t.Logf("Sepolia	Redeem session: %v", redeemSession)
	redeemSession, err = scalarClient.GetChainRedeemSession("evm|97", custodianGroupUid)
	if err != nil {
		t.Fatalf("Failed to get redeem session: %v", err)
	}
	t.Logf("BnbChain Redeem session: %v", redeemSession)
}
