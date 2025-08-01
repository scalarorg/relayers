package custodial_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/wire"
	vault "github.com/scalarorg/bitcoin-vault/ffi/go"
	"github.com/scalarorg/go-common/types"
	"github.com/stretchr/testify/require"
)

func SignPsbt(inputBytes []byte, privateKey string, finalize bool) ([]byte, error) {
	wif, err := btcutil.DecodeWIF(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode WIF: %w", err)
	}
	privKeyBytes := wif.PrivKey.Serialize()
	var networkKind types.NetworkKind
	// TODO: Cross-check with xchains core's btc module

	network := "testnet4"

	if network == "bitcoin" {
		networkKind = types.NetworkKindMainnet
	} else {
		networkKind = types.NetworkKindTestnet
	}
	partialSignedPsbt, err := vault.SignPsbtBySingleKey(
		inputBytes,      // []byte containing PSBT
		privKeyBytes[:], // []byte containing private key
		networkKind,     // bool indicating if testnet
		finalize,        // finalize
	)
	return partialSignedPsbt, nil
}

// Note: To run this test, must build bitcoin-vault-ffi first then copy to the lib folder
// cp ../../bitcoin-vault/target/release/libbitcoin_vault_ffi.* ./lib

// CGO_LDFLAGS="-L./lib -lbitcoin_vault_ffi" CGO_CFLAGS="-I./lib" go test -timeout 10m -run ^TestSignPsbt$ github.com/scalarorg/relayers/pkg/clients/custodial -v -count=1
var (
	mockWifs = []string{
		"cVszWmaCvZMdxgZFMTvfHvwwKBiYZHaAJ1NmUKnqqrXsqietjGtw", "cRu1NpeDu5zXRExEieQD78Juw1ACNtgLBR4Y7HLRVcgWf3Ym8ScP", "cTgLJaxyDwX8ZToE9jrJgoRneuZHhyHWM4k6F7HHSYZWG382Jwzq", "cVw4wnu15NXTT2zQzUVvgcVUmCMRGGgMDUuLUAvoJDdJvMEjk2j7", "cQj9m5WKR7rZH254UPfbcrh7EzN4PzLdasnRrSF7XLY9ZQyHzLid",
	}
	psbtBase64 = "cHNidP8BAFICAAAAAQ65NszjlFq/XK+8Mkb3R+UKvm6vr9TDmyjqKJGU1AzbAAAAAAD9////ARAnAAAAAAAAFgAUUNzsoVipyHLrQF1SKT01ERBXLJ4AAAAAAAEBK6CGAQAAAAAAIlEgf4Fav239eEI6cIqo2xwskG7srJEMA1Ey00LkmIo3uNUBAwQAAAAAIhXAUJKbdMGgSVS3i0tgNel6XgeKWg8o7JbVR7/ums6AOsCtIBXakTs+h7STKx4bh9lmfCjnJQqg7WCzoxCV9UHhZBSIrCBZTnjAopaCENnBVQ1K0xsD1eS5ZZzy9nhCSDuzwrt4EbogtZ5XXO+HPqlSc6/VWVbIRZBQcgDUEOaT5LB5pCbMYQK6IOLSJs/a7JOQPD87gaAagbGRN2J8sm5iGgr7e81u+8//uiDw89m+r3o5RbyqFH4EGuHVygKb3n5A2CUfB4PW7L6PtbpTosAhFhXakTs+h7STKx4bh9lmfCjnJQqg7WCzoxCV9UHhZBSIJQFaEKXscpYpxt2GPcKLcWLhj5awDe3YfxWLIoQoopi8ywAAAAAhFllOeMCiloIQ2cFVDUrTGwPV5LllnPL2eEJIO7PCu3gRJQFaEKXscpYpxt2GPcKLcWLhj5awDe3YfxWLIoQoopi8ywAAAAAhFrWeV1zvhz6pUnOv1VlWyEWQUHIA1BDmk+SweaQmzGECJQFaEKXscpYpxt2GPcKLcWLhj5awDe3YfxWLIoQoopi8ywAAAAAhFuLSJs/a7JOQPD87gaAagbGRN2J8sm5iGgr7e81u+8//JQFaEKXscpYpxt2GPcKLcWLhj5awDe3YfxWLIoQoopi8ywAAAAAhFvDz2b6vejlFvKoUfgQa4dXKApvefkDYJR8Hg9bsvo+1JQFaEKXscpYpxt2GPcKLcWLhj5awDe3YfxWLIoQoopi8ywAAAAABFyBQkpt0waBJVLeLS2A16XpeB4paDyjsltVHv+6azoA6wAEYIFoQpexylinG3YY9wotxYuGPlrAN7dh/FYsihCiimLzLAAA="
)

func TestSignPsbtBase64(t *testing.T) {
	packet, err := psbt.NewFromRawBytes(strings.NewReader(psbtBase64), true)
	require.NoError(t, err)
	var buf bytes.Buffer
	err = packet.Serialize(&buf)
	require.NoError(t, err)
	finalizedPsbt := buf.Bytes()
	t.Logf("psbt hex: %x", finalizedPsbt)
	for i, privateKey := range mockWifs {
		finalize := i == len(mockWifs)-1
		signedPsbt, err := SignPsbt(finalizedPsbt, privateKey, finalize)
		if err != nil {
			t.Fatalf("fail to sign psbt: %v", err)
		}
		finalizedPsbt = signedPsbt
	}
	finalTx := &wire.MsgTx{}
	err = finalTx.Deserialize(bytes.NewReader(finalizedPsbt))
	if err != nil {
		t.Fatalf("fail to deserialize psbt: %v", err)
	}
	finalTxBuf := bytes.NewBuffer(make([]byte, 0, finalTx.SerializeSize()))
	if err := finalTx.Serialize(finalTxBuf); err != nil {
		t.Fatalf("fail to serialize final tx: %v", err)
	}

	t.Logf("finalized psbt: %x", finalizedPsbt)
}
func TestSignPsbt(t *testing.T) {

	psbtHex := "70736274ff010052020000000160ababf6ca1e64e0940be05f368392f44ac8c9ecbf946baf802feb6fe3e9dd370000000000fdffffff01da0200000000000016001450dceca158a9c872eb405d52293d351110572c9e000000000001012be8030000000000002251207f815abf6dfd78423a708aa8db1c2c906eecac910c035132d342e4988a37b8d5010304000000002215c050929b74c1a04954b78b4b6035e97a5e078a5a0f28ec96d547bfee9ace803ac0ad2015da913b3e87b4932b1e1b87d9667c28e7250aa0ed60b3a31095f541e1641488ac20594e78c0a2968210d9c1550d4ad31b03d5e4b9659cf2f67842483bb3c2bb7811ba20b59e575cef873ea95273afd55956c84590507200d410e693e4b079a426cc6102ba20e2d226cfdaec93903c3f3b81a01a81b19137627cb26e621a0afb7bcd6efbcfffba20f0f3d9beaf7a3945bcaa147e041ae1d5ca029bde7e40d8251f0783d6ecbe8fb5ba53a2c0211615da913b3e87b4932b1e1b87d9667c28e7250aa0ed60b3a31095f541e164148825015a10a5ec729629c6dd863dc28b7162e18f96b00dedd87f158b228428a298bccb000000002116594e78c0a2968210d9c1550d4ad31b03d5e4b9659cf2f67842483bb3c2bb781125015a10a5ec729629c6dd863dc28b7162e18f96b00dedd87f158b228428a298bccb000000002116b59e575cef873ea95273afd55956c84590507200d410e693e4b079a426cc610225015a10a5ec729629c6dd863dc28b7162e18f96b00dedd87f158b228428a298bccb000000002116e2d226cfdaec93903c3f3b81a01a81b19137627cb26e621a0afb7bcd6efbcfff25015a10a5ec729629c6dd863dc28b7162e18f96b00dedd87f158b228428a298bccb000000002116f0f3d9beaf7a3945bcaa147e041ae1d5ca029bde7e40d8251f0783d6ecbe8fb525015a10a5ec729629c6dd863dc28b7162e18f96b00dedd87f158b228428a298bccb0000000001172050929b74c1a04954b78b4b6035e97a5e078a5a0f28ec96d547bfee9ace803ac00118205a10a5ec729629c6dd863dc28b7162e18f96b00dedd87f158b228428a298bccb0000"

	finalizedPsbt, err := hex.DecodeString(psbtHex)
	if err != nil {
		t.Fatalf("fail to decode psbt hex: %v", err)
	}

	t.Logf("psbt hex: %x", finalizedPsbt)
	for i, privateKey := range mockWifs {
		finalize := i == len(mockWifs)-1
		signedPsbt, err := SignPsbt(finalizedPsbt, privateKey, finalize)
		if err != nil {
			t.Fatalf("fail to sign psbt: %v", err)
		}
		finalizedPsbt = signedPsbt
	}
	finalTx := &wire.MsgTx{}
	err = finalTx.Deserialize(bytes.NewReader(finalizedPsbt))
	if err != nil {
		t.Fatalf("fail to deserialize psbt: %v", err)
	}
	finalTxBuf := bytes.NewBuffer(make([]byte, 0, finalTx.SerializeSize()))
	if err := finalTx.Serialize(finalTxBuf); err != nil {
		t.Fatalf("fail to serialize final tx: %v", err)
	}

	t.Logf("finalized psbt: %x", finalizedPsbt)
}

// 70736274ff010052020000000160ababf6ca1e64e0940be05f368392f44ac8c9ecbf946baf802feb6fe3e9dd370000000000fdffffff01da0200000000000016001450dceca158a9c872eb405d52293d351110572c9e000000000001012be8030000000000002251207f815abf6dfd78423a708aa8db1c2c906eecac910c035132d342e4988a37b8d52215c050929b74c1a04954b78b4b6035e97a5e078a5a0f28ec96d547bfee9ace803ac0ad2015da913b3e87b4932b1e1b87d9667c28e7250aa0ed60b3a31095f541e1641488ac20594e78c0a2968210d9c1550d4ad31b03d5e4b9659cf2f67842483bb3c2bb7811ba20b59e575cef873ea95273afd55956c84590507200d410e693e4b079a426cc6102ba20e2d226cfdaec93903c3f3b81a01a81b19137627cb26e621a0afb7bcd6efbcfffba20f0f3d9beaf7a3945bcaa147e041ae1d5ca029bde7e40d8251f0783d6ecbe8fb5ba53a2c0211615da913b3e87b4932b1e1b87d9667c28e7250aa0ed60b3a31095f541e164148825015a10a5ec729629c6dd863dc28b7162e18f96b00dedd87f158b228428a298bccb000000002116594e78c0a2968210d9c1550d4ad31b03d5e4b9659cf2f67842483bb3c2bb781125015a10a5ec729629c6dd863dc28b7162e18f96b00dedd87f158b228428a298bccb000000002116b59e575cef873ea95273afd55956c84590507200d410e693e4b079a426cc610225015a10a5ec729629c6dd863dc28b7162e18f96b00dedd87f158b228428a298bccb000000002116e2d226cfdaec93903c3f3b81a01a81b19137627cb26e621a0afb7bcd6efbcfff25015a10a5ec729629c6dd863dc28b7162e18f96b00dedd87f158b228428a298bccb000000002116f0f3d9beaf7a3945bcaa147e041ae1d5ca029bde7e40d8251f0783d6ecbe8fb525015a10a5ec729629c6dd863dc28b7162e18f96b00dedd87f158b228428a298bccb0000000001172050929b74c1a04954b78b4b6035e97a5e078a5a0f28ec96d547bfee9ace803ac00118205a10a5ec729629c6dd863dc28b7162e18f96b00dedd87f158b228428a298bccb0000

// 0200000000010160ababf6ca1e64e0940be05f368392f44ac8c9ecbf946baf802feb6fe3e9dd370000000000fdffffff01da0200000000000016001450dceca158a9c872eb405d52293d351110572c9e0740428565132877b2fb596970847518832ff3b48afe4c6d05c8943c025ebf51b1a000f74c2b6164598deb0707d1b50bbfe73aedbd7d4d047cd0812d3d9749ef566c40549aea717d1dc15a507e24c2ba111396053d622d8d9f59d3f3760d2e01a9d9476c00fde17361411557f432125f2655fa607dd78d36c399810a984eaacd0f578840dde29802ed6e1fc7a1cebe00f22d20defb8d4d2621772fb53268bf7bb9f14b920d93656447ed2a0ee26698e9fb84bed1b5101a95cc15b73b83f0e386e349a6b00000ac2015da913b3e87b4932b1e1b87d9667c28e7250aa0ed60b3a31095f541e1641488ac20594e78c0a2968210d9c1550d4ad31b03d5e4b9659cf2f67842483bb3c2bb7811ba20b59e575cef873ea95273afd55956c84590507200d410e693e4b079a426cc6102ba20e2d226cfdaec93903c3f3b81a01a81b19137627cb26e621a0afb7bcd6efbcfffba20f0f3d9beaf7a3945bcaa147e041ae1d5ca029bde7e40d8251f0783d6ecbe8fb5ba53a221c050929b74c1a04954b78b4b6035e97a5e078a5a0f28ec96d547bfee9ace803ac00000000

// 0200000000010160ababf6ca1e64e0940be05f368392f44ac8c9ecbf946baf802feb6fe3e9dd370000000000fdffffff01da0200000000000016001450dceca158a9c872eb405d52293d351110572c9e0740428565132877b2fb596970847518832ff3b48afe4c6d05c8943c025ebf51b1a000f74c2b6164598deb0707d1b50bbfe73aedbd7d4d047cd0812d3d9749ef566c40549aea717d1dc15a507e24c2ba111396053d622d8d9f59d3f3760d2e01a9d9476c00fde17361411557f432125f2655fa607dd78d36c399810a984eaacd0f578840dde29802ed6e1fc7a1cebe00f22d20defb8d4d2621772fb53268bf7bb9f14b920d93656447ed2a0ee26698e9fb84bed1b5101a95cc15b73b83f0e386e349a6b040679a383a56e747e99f206b6ad9618c7523c3fd66f9e7acadddc0ab90536756756c574173826d77e4a8dcb77d9201ff50223843eb1c5acf1788ee5adcc63930dd40151a53cc377916092ef06890e698d8afd5fb65cdb7cb15e2eb6356ca85e58446d1ca8319a12bc5b9cc6f585f7ba5b48783e0554fc7e506c03f0da79c1814e03fac2015da913b3e87b4932b1e1b87d9667c28e7250aa0ed60b3a31095f541e1641488ac20594e78c0a2968210d9c1550d4ad31b03d5e4b9659cf2f67842483bb3c2bb7811ba20b59e575cef873ea95273afd55956c84590507200d410e693e4b079a426cc6102ba20e2d226cfdaec93903c3f3b81a01a81b19137627cb26e621a0afb7bcd6efbcfffba20f0f3d9beaf7a3945bcaa147e041ae1d5ca029bde7e40d8251f0783d6ecbe8fb5ba53a221c050929b74c1a04954b78b4b6035e97a5e078a5a0f28ec96d547bfee9ace803ac00000000
