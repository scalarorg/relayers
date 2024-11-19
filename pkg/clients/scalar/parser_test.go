package scalar_test

import (
	"fmt"
	"testing"

	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/stretchr/testify/require"
)

func TestDecodeHash(t *testing.T) {
	payloadHash := "[44,203,134,96,61,64,78,231,161,152,45,125,221,206,36,22,170,30,51,72,8,2,126,133,165,116,253,75,235,100,45,69]"
	fmt.Println(payloadHash)
	decoded, err := scalar.DecodeIntArrayToHexString(payloadHash)
	require.NoError(t, err)
	require.Equal(t, "2ccb86603d404ee7a1982d7dddce2416aa1e334808027e85a574fd4beb642d45", decoded)
	fmt.Println(decoded)
}
