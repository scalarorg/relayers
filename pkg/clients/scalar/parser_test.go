package scalar_test

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/scalarorg/scalar-core/x/chains/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	//"google.golang.org/protobuf/proto"
)

var (
	jsonData = `{
	"coin_received.amount":	["185625002719122655546ascal","185625035952687993072ascal","185625035952687993072ascal"],
	"coin_received.receiver":["scalar1jv65s3grqf6v6jl3dp4t6c9t9rk99cd84dc5vq","scalar1m3h30wlvsf8llruxtpukdvsy0km2kum86rmszl","scalar17xpfvakm2amg962yls6f84z3kell8c5lztlgwz"],
	"coin_spent.amount":["185625002719122655546ascal","185625035952687993072ascal"],
	"coin_spent.spender":["scalar17xpfvakm2amg962yls6f84z3kell8c5lztlgwz","scalar1m3h30wlvsf8llruxtpukdvsy0km2kum86rmszl"],
	"coinbase.amount":["185625035952687993072ascal"],
	"coinbase.minter":["scalar1m3h30wlvsf8llruxtpukdvsy0km2kum86rmszl"],
	"commission.amount":["9281250135956132777.300000000000000000ascal","92070001348684837093.272249157071976780ascal","51789375758635220897.334000000000000000ascal","23017500337171209230.160249157071976780ascal","5754375084292802264.382249157071976780ascal"],
	"commission.validator":["scalarvaloper105wsknjutftqksqtugr83avrlsxdz0tvc96yhk","scalarvaloper105wsknjutftqksqtugr83avrlsxdz0tvc96yhk","scalarvaloper1tc9q2ngj29kllv4kfrqjh6gj4hrsp6p35zpyep","scalarvaloper1qyhnraq6dwpl69heem9qd4phrd946lct6xdgqg","scalarvaloper1u8ennvwsshneu4nvec38e3jvcxmppf7lfq3pf5"],
	"message.sender":["scalar17xpfvakm2amg962yls6f84z3kell8c5lztlgwz","scalar1m3h30wlvsf8llruxtpukdvsy0km2kum86rmszl"],
	"mint.amount":["185625035952687993072"],
	"mint.annual_provisions":["1171576126916109322037617349.891394409147926186"],
	"mint.bonded_ratio":["0.000000000000003328"],"mint.inflation":["0.130001585988795083"],
	"proposer_reward.amount":["9281250135956132777.300000000000000000ascal"],
	"proposer_reward.validator":["scalarvaloper105wsknjutftqksqtugr83avrlsxdz0tvc96yhk"],
	"rewards.amount":["9281250135956132777.300000000000000000ascal","92070001348684837093.272249157071976780ascal","51789375758635220897.334000000000000000ascal","23017500337171209230.160249157071976780ascal","5754375084292802264.382249157071976780ascal"],
	"rewards.validator":["scalarvaloper105wsknjutftqksqtugr83avrlsxdz0tvc96yhk","scalarvaloper105wsknjutftqksqtugr83avrlsxdz0tvc96yhk","scalarvaloper1tc9q2ngj29kllv4kfrqjh6gj4hrsp6p35zpyep","scalarvaloper1qyhnraq6dwpl69heem9qd4phrd946lct6xdgqg","scalarvaloper1u8ennvwsshneu4nvec38e3jvcxmppf7lfq3pf5"],
	"scalar.chains.v1beta1.ContractCallApproved.chain":["\"evm|11155111\""],
	"scalar.chains.v1beta1.ContractCallApproved.command_id":["[216,24,192,178,234,134,203,174,53,152,189,26,210,97,212,0,37,25,215,39,92,169,55,110,150,157,109,194,170,153,170,16]"],
	"scalar.chains.v1beta1.ContractCallApproved.contract_address":["\"0x0000000000000000000000000000000000000000\""],
	"scalar.chains.v1beta1.ContractCallApproved.destination_chain":["\"bitcoin|4\""],
	"scalar.chains.v1beta1.ContractCallApproved.event_id":["\"0xcabc3140a564038f9f76e9d692309ea8d5be7d5a8a2133b97bf0579a73cfbb37-288\""],
	"scalar.chains.v1beta1.ContractCallApproved.payload_hash":["[163,161,152,24,44,45,10,135,85,128,20,128,254,13,202,225,73,27,188,103,173,247,208,12,183,71,67,208,196,81,7,170]"],
	"scalar.chains.v1beta1.ContractCallApproved.sender":["\"0x982321eb5693cdbAadFfe97056BEce07D09Ba49f\""],
	"scalar.nexus.v1beta1.MessageExecuted.destination_chain":["\"bitcoin|4\""],
	"scalar.nexus.v1beta1.MessageExecuted.id":["\"0xcabc3140a564038f9f76e9d692309ea8d5be7d5a8a2133b97bf0579a73cfbb37-288\""],
	"scalar.nexus.v1beta1.MessageExecuted.source_chain":["\"evm|11155111\""],
	"tm.event":["NewBlock"],
	"transfer.amount":["185625002719122655546ascal","185625035952687993072ascal"],
	"transfer.recipient":["scalar1jv65s3grqf6v6jl3dp4t6c9t9rk99cd84dc5vq","scalar17xpfvakm2amg962yls6f84z3kell8c5lztlgwz"],
	"transfer.sender":["scalar17xpfvakm2amg962yls6f84z3kell8c5lztlgwz","scalar1m3h30wlvsf8llruxtpukdvsy0km2kum86rmszl"]
}`
	EVENT_TOKEN_SENT = `{
	"asset":"{\"denom\":\"pBtc\",\"amount\":\"10000\"}",
	"chain":"\"evm|11155111\"",
	"destination_address":"\"0x982321eb5693cdbAadFfe97056BEce07D09Ba49f\"",
	"destination_chain":"\"evm|97\"",
	"event_id":"\"0x620bc60a616248eaf0a9f5b7e45db3f96eca31420c581034a6c59669cefb7de1-240\""
	,"sender":"\"0x982321eb5693cdbAadFfe97056BEce07D09Ba49f\"",
	"transfer_id":"\"0\""}`
	EVENT_RAW = `
	{"coin_received.amount":"185624770084218383084ascal",
	"coin_received.receiver":"scalar1jv65s3grqf6v6jl3dp4t6c9t9rk99cd84dc5vq",
	"coin_spent.amount":"185624770084218383084ascal",
	"coin_spent.spender":"scalar17xpfvakm2amg962yls6f84z3kell8c5lztlgwz",
	"coinbase.amount":"185624803317770448058ascal",
	"coinbase.minter":"scalar1m3h30wlvsf8llruxtpukdvsy0km2kum86rmszl",
	"commission.amount":"9281238504210919154.200000000000000000ascal",
	"commission.validator":"scalarvaloper105wsknjutftqksqtugr83avrlsxdz0tvc96yhk",
	"message.sender":"scalar17xpfvakm2amg962yls6f84z3kell8c5lztlgwz",
	"mint.amount":"185624803317770448058",
	"mint.annual_provisions":"1171574658636174538330848875.072530810640694180",
	"mint.bonded_ratio":"0.000000000000003328",
	"mint.inflation":"0.130001441807995530",
	"proposer_reward.amount":"9281238504210919154.200000000000000000ascal",
	"proposer_reward.validator":"scalarvaloper105wsknjutftqksqtugr83avrlsxdz0tvc96yhk",
	"rewards.amount":"9281238504210919154.200000000000000000ascal",
	"rewards.validator":"scalarvaloper105wsknjutftqksqtugr83avrlsxdz0tvc96yhk",
	"scalar.chains.v1beta1.ChainEventCompleted.chain":"\"evm|11155111\"",
	"scalar.chains.v1beta1.ChainEventCompleted.event_id":"\"0x00f66cd43ed1ae4480797100a157fcc6e153d9c1deb953001ba10797eb2841d2-114\"",
	"scalar.chains.v1beta1.ChainEventCompleted.type":"\"Event_TokenSent\"",
	"scalar.chains.v1beta1.EventTokenSent.asset":"{\"denom\":\"pBtc\",\"amount\":\"10000\"}",
	"scalar.chains.v1beta1.EventTokenSent.chain":"\"evm|11155111\"",
	"scalar.chains.v1beta1.EventTokenSent.destination_address":"\"0x982321eb5693cdbAadFfe97056BEce07D09Ba49f\"",
	"scalar.chains.v1beta1.EventTokenSent.destination_chain":"\"evm|97\"",
	"scalar.chains.v1beta1.EventTokenSent.event_id":"\"0x00f66cd43ed1ae4480797100a157fcc6e153d9c1deb953001ba10797eb2841d2-114\"",
	"scalar.chains.v1beta1.EventTokenSent.sender":"\"0x982321eb5693cdbAadFfe97056BEce07D09Ba49f\"",
	"scalar.chains.v1beta1.EventTokenSent.transfer_id":"\"1\"",
	"scalar.nexus.v1beta1.FeeDeducted.amount":"{\"denom\":\"pBtc\",\"amount\":\"20000\"}",
	"scalar.nexus.v1beta1.FeeDeducted.fee":"{\"denom\":\"pBtc\",\"amount\":\"0\"}",
	"scalar.nexus.v1beta1.FeeDeducted.recipient_address":"\"0x982321eb5693cdbAadFfe97056BEce07D09Ba49f\"",
	"scalar.nexus.v1beta1.FeeDeducted.recipient_chain":"\"evm|97\"",
	"scalar.nexus.v1beta1.FeeDeducted.transfer_id":"\"1\"",
	"tm.event":"NewBlock",
	"transfer.amount":"185624770084218383084ascal",
	"transfer.recipient":"scalar1jv65s3grqf6v6jl3dp4t6c9t9rk99cd84dc5vq",
	"transfer.sender":"scalar17xpfvakm2amg962yls6f84z3kell8c5lztlgwz"}
	`
)

func TestDecodeHash(t *testing.T) {
	payloadHash := "[44,203,134,96,61,64,78,231,161,152,45,125,221,206,36,22,170,30,51,72,8,2,126,133,165,116,253,75,235,100,45,69]"
	fmt.Println(payloadHash)
	decoded, err := scalar.DecodeIntArrayToHexString(payloadHash)
	require.NoError(t, err)
	require.Equal(t, "2ccb86603d404ee7a1982d7dddce2416aa1e334808027e85a574fd4beb642d45", decoded)
	fmt.Println(decoded)
}
func TestParseTokenSent(t *testing.T) {
	var tokenSent types.EventTokenSent
	var rawMap map[string]string
	if err := json.Unmarshal([]byte(EVENT_TOKEN_SENT), &rawMap); err != nil {
		log.Fatalf("Error unmarshalling to map: %v", err)
	}
	t.Logf("RawMap %v\n", rawMap)
	err := scalar.UnmarshalTokenSent(rawMap, &tokenSent)
	assert.NoError(t, err)
	t.Logf("Parsed event %v\n", tokenSent)
}

func TestParseContractCallApprovedEvent(t *testing.T) {
	// Map to hold the raw JSON
	var rawMap map[string][]string
	if err := json.Unmarshal([]byte(jsonData), &rawMap); err != nil {
		log.Fatalf("Error unmarshalling to map: %v", err)
	}
	event := types.ContractCallApproved{}
	msgName := proto.MessageName(&event)
	assert.Equal(t, "scalar.chains.v1beta1.ContractCallApproved", msgName)
	// targets := map[string]reflect.Type{"": }
	events, err := scalar.ParseIBCEvent[*types.ContractCallApproved](rawMap)
	assert.NoError(t, err)
	fmt.Printf("Events %v\n", events)
	//fmt.Printf("Raw Map %v:", rawMap)
}

func TestParseIBCEvent(t *testing.T) {
	// Map to hold the raw JSON
	var rawMap map[string][]string
	if err := json.Unmarshal([]byte(EVENT_RAW), &rawMap); err != nil {
		log.Fatalf("Error unmarshalling to map: %v", err)
	}
	fmt.Printf("Raw Map %v:", rawMap)
	// event := types.ContractCallApproved
	// msgName := proto.MessageName(&event)
	// fmt.Printf("%s", msgName)
	// targets := map[string]reflect.Type{"": }
	events, err := scalar.ParseIBCEvent[*types.EventTokenSent](rawMap)
	assert.NoError(t, err)
	fmt.Printf("Events %v", events)

}
