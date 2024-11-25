package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindPayloadByHash(t *testing.T) {
	payload, err := dbAdapter.FindPayloadByHash("0x123456789abcdef")
	require.NoError(t, err)
	require.NotNil(t, payload)
}
