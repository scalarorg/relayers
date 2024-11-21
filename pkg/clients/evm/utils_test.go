package evm_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/scalarorg/relayers/pkg/clients/evm"
	"github.com/stretchr/testify/assert"
)

var (
	executeData1 = "09c5eabe000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000008c00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000058000000000000000000000000000000000000000000000000000000000000005200000000000000000000000000000000000000000000000000000000000aa36a7000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000e000000000000000000000000000000000000000000000000000000000000001c000000000000000000000000000000000000000000000000000000000000000024b6f70ea9c9556453ab0f8b65003118239719ae504bd19559f21e0f3c5021479bac7d491529a703e5af3336c32ff64b73b0cd0ed25cc61a43567b25741dde8b60000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000013617070726f7665436f6e747261637443616c6c000000000000000000000000000000000000000000000000000000000000000000000000000000000000000013617070726f7665436f6e747261637443616c6c000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000001c0000000000000000000000000000000000000000000000000000000000000016000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001000000000000000000000000001f98c06d8734d5a9ff0b53e3294626e62e4d232c2ccb86603d404ee7a1982d7dddce2416aa1e334808027e85a574fd4beb642d457c1b7e6978cb930a0e42b632773e564be64eb64da491aee447a4fa836ead8a0400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010626974636f696e2d746573746e65743400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002a30783133304334383130443537313430653145363239363763424637343243614561453931623665634500000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000016000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001000000000000000000000000001f98c06d8734d5a9ff0b53e3294626e62e4d232c70859da59f66b872df9495245c3f69a35137d3a7b7ef7b255557e3d3065c95a3a1c6f8acb05f02085139c3cd4a16c18edc55fa36cc49551c4caf611ce3f37daa00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010626974636f696e2d746573746e65743400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002a30783133304334383130443537313430653145363239363763424637343243614561453931623665634500000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000000000ea6000000000000000000000000000000000000000000000000000000000000001c00000000000000000000000000000000000000000000000000000000000000004000000000000000000000000475bd32787373ab356f5173c43860ee84c52d9bb0000000000000000000000006bac56260cc1f884ead16213645b7b57b57593a90000000000000000000000008ee0054a5a803e37d7773892bcdcc9bafeb766d7000000000000000000000000c45cffd93ab6d2c7229b4e3c4ce46fe242aa740f00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000004e20000000000000000000000000000000000000000000000000000000000000753000000000000000000000000000000000000000000000000000000000000027100000000000000000000000000000000000000000000000000000000000009c400000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000000417f5509eebd9222503b3ade3e0cb7b508a46eb700e2057e7887d0de00d3e2cc3c72cd359535cf67cd59592e0fc976578794bd5395092694e4034023493a22987e1c000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000041b086cf93f7957a084380f7e911cc6e28885eec7c6132c3d19a865ffc2f6188991bacb2ca81c66bbd294e9372d6bd82a6841a833a4ec6590270fd8e6c79aa6bf81b00000000000000000000000000000000000000000000000000000000000000"
	executeData2 = "09c5eabe000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000006a00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000036000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000011232a596e9a96bdea93cecf444ecd9a3ec227d278f6dd8ef346758d2d0f54229000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000013617070726f7665436f6e747261637443616c6c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000016000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000b5ee0d2ebb24841fbadc76a6196cd01f78903b636404b3cb3f3a521527a4361a91cc3c9cfdab0523d5217430273cd9dde9abe9afbb243ff34c94f3af9a43b2de00000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000010657468657265756d2d7365706f6c696100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002a30784239316533413845663836323536373032366436463337366339463364366238313443613433333700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000000000ea6000000000000000000000000000000000000000000000000000000000000001c00000000000000000000000000000000000000000000000000000000000000004000000000000000000000000057cc688a7a0336897a315e38643e42ec207e2090000000000000000000000000e942c5e30adf86f844339850c5791355efd3d95000000000000000000000000a2ffff8288c4ee3bce7b26816fc606b403aca185000000000000000000000000bc57d589b69d26d6e64a9aec17eb35fbb5b158bc00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000009c400000000000000000000000000000000000000000000000000000000000004e20000000000000000000000000000000000000000000000000000000000000271000000000000000000000000000000000000000000000000000000000000075300000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000000413f339fb7bcef84ae966ddd7eac39bd56934ffe438613728d69c93168633f5d537b2fc29f17e25e2e84da7b702081bc45e97fe2dc8d58be8150585a15a56a56eb1b000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000041ac92856c6b90a7df861c466fd0b86298fa544af1a36b173df6c2737eaaa2125d64599e6c9df173a2e9dff2331aa2034d0ba742334dbf83c4bed19cccd48889431b00000000000000000000000000000000000000000000000000000000000000"
)

func TestDecodeExecuteData1(t *testing.T) {
	decodedExecuteData, err := evm.DecodeExecuteData(executeData1)
	assert.Nil(t, err)
	commandIdHexs := make([]string, len(decodedExecuteData.CommandIds))
	for i, commandId := range decodedExecuteData.CommandIds {
		commandIdHexs[i] = hex.EncodeToString(commandId[:])
	}
	expectedCommandIds := make([]string, len(commandIdHexs))
	expectedCommandIds[0] = "4b6f70ea9c9556453ab0f8b65003118239719ae504bd19559f21e0f3c5021479"
	expectedCommandIds[1] = "bac7d491529a703e5af3336c32ff64b73b0cd0ed25cc61a43567b25741dde8b6"
	t.Logf("commandIdHexs: %v", commandIdHexs)
	t.Logf("commands: %v", decodedExecuteData.Commands)
	assert.NotNil(t, decodedExecuteData)
	assert.Equal(t, uint64(0xaa36a7), decodedExecuteData.ChainId)
	//assert.Equal(t, decodedExecuteData.CommandIds, expectedCommandIds)
	assert.Equal(t, []string{"approveContractCall", "approveContractCall"}, decodedExecuteData.Commands)
	assert.Equal(t, len(decodedExecuteData.CommandIds), len(decodedExecuteData.Params))
	assert.Equal(t, []uint64{20000, 30000, 10000, 40000}, decodedExecuteData.Weights)
	assert.Equal(t, uint64(60000), decodedExecuteData.Threshold)
	assert.Equal(t, []string{"7f5509eebd9222503b3ade3e0cb7b508a46eb700e2057e7887d0de00d3e2cc3c72cd359535cf67cd59592e0fc976578794bd5395092694e4034023493a22987e1c",
		"b086cf93f7957a084380f7e911cc6e28885eec7c6132c3d19a865ffc2f6188991bacb2ca81c66bbd294e9372d6bd82a6841a833a4ec6590270fd8e6c79aa6bf81b"},
		decodedExecuteData.Signatures)
	t.Logf("decodedExecuteData: %v", decodedExecuteData)
	fmt.Printf("decodedExecuteData: %v", decodedExecuteData)
}

func TestDecodeExecuteData2(t *testing.T) {
	decodedExecuteData, _ := evm.DecodeExecuteData(executeData2)
	expectedCommandId, err := hex.DecodeString("f68541762276895b200060168ccdf90abb8c621b0229739b9f1dc9b1903e20aa")
	assert.Nil(t, err)
	assert.NotNil(t, decodedExecuteData)
	assert.Equal(t, decodedExecuteData.ChainId, uint64(11155111))
	assert.Equal(t, decodedExecuteData.CommandIds, [][32]byte{[32]byte(expectedCommandId)})
	assert.Equal(t, decodedExecuteData.Commands, []string{"approveContractCall"})
	assert.Equal(t, len(decodedExecuteData.Params), 1)
	assert.Equal(t, decodedExecuteData.Weights, []uint64{20000, 30000, 10000, 40000})
	assert.Equal(t, decodedExecuteData.Threshold, uint64(60000))
	assert.Equal(t, decodedExecuteData.Signatures, []string{"87c6ff258a853893ffb33fafa25d5fd18e40b98d6a40dc6ad3def069f2025905146d4277dfef5900489e23f071ef1e8e303e872154c5f2a7dbbf3559a544174e1c", "bcf84bfcbb26c5055ba5e3d542bbdca32c7b8444056c5d3c8a18556840394f0771b3fb2b9e064ffedd00bca2b9326c61a9e05a47594a031901c4b63b67ab5af71c"})
	t.Logf("decodedExecuteData: %v", decodedExecuteData)
	fmt.Printf("decodedExecuteData: %v", decodedExecuteData)
}
