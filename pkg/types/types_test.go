package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSessionCmp(t *testing.T) {
	session1 := &Session{Sequence: 1, Phase: 0}
	session2 := &Session{Sequence: 1, Phase: 1}
	session3 := &Session{Sequence: 2, Phase: 0}
	session4 := &Session{Sequence: 2, Phase: 1}

	res21 := session2.Cmp(session1)
	require.Equal(t, int64(1), res21)
	res34 := session3.Cmp(session4)
	require.Equal(t, int64(-1), res34)
	res43 := session4.Cmp(session3)
	require.Equal(t, int64(1), res43)
	res12 := session1.Cmp(session2)
	require.Equal(t, int64(-1), res12)
	res13 := session1.Cmp(session3)
	require.Equal(t, int64(-2), res13)
	res41 := session4.Cmp(session1)
	require.Equal(t, int64(3), res41)
}
