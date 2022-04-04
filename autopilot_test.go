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

func TestDisabledReconcilation(t *testing.T) {
	logger := testLogger(t)
	ap := New(NewMockRaft(t), NewMockApplicationIntegration(t), WithLogger(logger), WithReconciliationDisabled())
	require.False(t, ap.ReconciliationEnabled())

	ap.EnableReconciliation()
	require.True(t, ap.ReconciliationEnabled())

	ap.DisableReconciliation()
	require.False(t, ap.ReconciliationEnabled())

	ap = New(NewMockRaft(t), NewMockApplicationIntegration(t), WithLogger(logger))
	require.True(t, ap.ReconciliationEnabled())
}
