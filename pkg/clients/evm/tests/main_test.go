package evm_test

import (
	"os"
	"testing"
)

var (
	TEST_RPC_ENDPOINT     = "ws://localhost:8546/"
	TEST_CONTRACT_ADDRESS = "0x2bb588d7bb6faAA93f656C3C78fFc1bEAfd1813D"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
