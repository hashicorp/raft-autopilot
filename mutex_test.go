package autopilot

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMutex(t *testing.T) {
	t.Run("starts unlocked", func(t *testing.T) {
		m := newMutex()
		canLock(t, m)
	})

	t.Run("Lock blocks when locked", func(t *testing.T) {
		m := newMutex()
		m.Lock()
		lockIsBlocked(t, m)
		// needed to unblock the go routine started by lockIsBlocked
		// in order to prevent leaking the go routine.
		m.Unlock()
	})

	t.Run("Unlock unblocks Lock", func(t *testing.T) {
		m := newMutex()
		m.Lock()
		m.Unlock() // nolint:staticcheck // SA2001 is not relevant here
		canLock(t, m)
	})

	t.Run("TryLock acquires lock", func(t *testing.T) {
		m := newMutex()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		t.Cleanup(cancel)
		require.NoError(t, m.TryLock(ctx))
		lockIsBlocked(t, m)
		// needed to unblock the go routine and prevent a leak
		m.Unlock()
	})

	t.Run("TryLock blocks until timeout when locked", func(t *testing.T) {
		m := newMutex()
		m.Lock()

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		t.Cleanup(cancel)
		err := m.TryLock(ctx)
		require.Equal(t, err, context.DeadlineExceeded)
	})

	t.Run("TryLock acquires lock before timeout", func(t *testing.T) {
		m := newMutex()
		m.Lock()

		go func() {
			time.Sleep(20 * time.Millisecond)
			m.Unlock()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		t.Cleanup(cancel)
		err := m.TryLock(ctx)
		require.NoError(t, err)
	})

}

func canLock(t *testing.T, m *mutex) {
	t.Helper()
	chDone := make(chan struct{})
	go func() {
		m.Lock()
		close(chDone)
	}()

	select {
	case <-chDone:
	case <-time.After(20 * time.Millisecond):
		t.Fatal("failed to acquire lock before timeout")
	}
}

func lockIsBlocked(t *testing.T, m *mutex) {
	t.Helper()
	chDone := make(chan struct{})
	go func() {
		m.Lock()
		close(chDone)
	}()

	select {
	case <-chDone:
		t.Fatal("expected Lock to block")
	case <-time.After(20 * time.Millisecond):
	}
}
