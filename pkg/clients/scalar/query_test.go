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
