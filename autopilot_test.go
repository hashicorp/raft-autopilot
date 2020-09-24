package autopilot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func chanIsSelectable(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestRemoveDeadServerTrigger(t *testing.T) {
	ap := New(NewMockRaft(t), NewMockApplicationIntegration(t))

	ap.RemoveDeadServers()

	require.True(t, chanIsSelectable(ap.removeDeadCh))
}
