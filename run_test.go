package autopilot

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go.uber.org/goleak"
)

type testWriter struct {
	t *testing.T
}

func (tw testWriter) Write(p []byte) (n int, err error) {
	tw.t.Log(strings.Trim(string(p), "\n"))
	return len(p), nil
}

func testLogger(t *testing.T) hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Name:        t.Name(),
		Level:       hclog.Trace,
		Output:      testWriter{t: t},
		DisableTime: true,
	})
}

func TestRunLifeCycle(t *testing.T) {
	// ensure that the code was honest and reported things as finished when the go routines
	// had gotten shut down
	t.Cleanup(func() { goleak.VerifyNone(t) })

	mraft := NewMockRaft(t)
	mdel := NewMockApplicationIntegration(t)
	mtime := NewMockTimeProvider(t)

	conf := &Config{
		CleanupDeadServers:      true,
		LastContactThreshold:    200 * time.Millisecond,
		MaxTrailingLogs:         200,
		MinQuorum:               3,
		ServerStabilizationTime: 10 * time.Second,
	}

	servers := map[raft.ServerID]*Server{
		"7875975d-d54b-49c1-a400-9fefcc706c67": {
			ID:          "7875975d-d54b-49c1-a400-9fefcc706c67",
			Name:        "node1",
			Address:     "198.18.0.1:8300",
			NodeStatus:  NodeAlive,
			Version:     "1.9.0",
			RaftVersion: 3,
		},
		"ecfc5237-63c3-4b09-94b9-d5682d9ae5b1": {
			ID:          "ecfc5237-63c3-4b09-94b9-d5682d9ae5b1",
			Name:        "node2",
			Address:     "198.18.0.2:8300",
			NodeStatus:  NodeAlive,
			Version:     "1.9.0",
			RaftVersion: 3,
		},
		"e72eb8da-604d-47cd-bd7f-69ec120ea2b7": {
			ID:          "e72eb8da-604d-47cd-bd7f-69ec120ea2b7",
			Name:        "node3",
			Address:     "198.18.0.3:8300",
			NodeStatus:  NodeAlive,
			Version:     "1.9.0",
			RaftVersion: 3,
		},
	}
	var lastIndex uint64 = 1024
	var lastTerm uint64 = 3

	serverStats := map[raft.ServerID]*ServerStats{
		"7875975d-d54b-49c1-a400-9fefcc706c67": {
			LastTerm:  lastTerm,
			LastIndex: lastIndex,
		},
		"ecfc5237-63c3-4b09-94b9-d5682d9ae5b1": {
			LastContact: 10 * time.Millisecond,
			LastTerm:    lastTerm,
			LastIndex:   1000,
		},
		"e72eb8da-604d-47cd-bd7f-69ec120ea2b7": {
			LastContact: 15 * time.Millisecond,
			LastTerm:    lastTerm,
			LastIndex:   999,
		},
	}

	var leaderAddr raft.ServerAddress = "198.18.0.1:8300"

	firstStateTime := time.Date(2020, 11, 2, 12, 0, 0, 0, time.UTC)
	restartStateTime := time.Date(2020, 11, 2, 12, 1, 0, 0, time.UTC)

	mtime.On("Now").Return(firstStateTime).Once()
	mtime.On("Now").Return(restartStateTime).Once()

	// now validate the initial state
	genExpected := func(ts time.Time) *State {
		return &State{
			firstStateTime:   ts,
			Healthy:          true,
			FailureTolerance: 1,
			Servers: map[raft.ServerID]*ServerState{
				"7875975d-d54b-49c1-a400-9fefcc706c67": {
					Server: Server{
						ID:          "7875975d-d54b-49c1-a400-9fefcc706c67",
						Name:        "node1",
						Address:     "198.18.0.1:8300",
						NodeStatus:  NodeAlive,
						Version:     "1.9.0",
						RaftVersion: 3,
						NodeType:    NodeVoter,
					},
					State:  RaftLeader,
					Stats:  *serverStats["7875975d-d54b-49c1-a400-9fefcc706c67"],
					Health: ServerHealth{Healthy: true, StableSince: ts},
				},
				"ecfc5237-63c3-4b09-94b9-d5682d9ae5b1": {
					Server: Server{
						ID:          "ecfc5237-63c3-4b09-94b9-d5682d9ae5b1",
						Name:        "node2",
						Address:     "198.18.0.2:8300",
						NodeStatus:  NodeAlive,
						Version:     "1.9.0",
						RaftVersion: 3,
						NodeType:    NodeVoter,
					},
					State:  RaftVoter,
					Stats:  *serverStats["ecfc5237-63c3-4b09-94b9-d5682d9ae5b1"],
					Health: ServerHealth{Healthy: true, StableSince: ts},
				},
				"e72eb8da-604d-47cd-bd7f-69ec120ea2b7": {
					Server: Server{
						ID:          "e72eb8da-604d-47cd-bd7f-69ec120ea2b7",
						Name:        "node3",
						Address:     "198.18.0.3:8300",
						NodeStatus:  NodeAlive,
						Version:     "1.9.0",
						RaftVersion: 3,
						NodeType:    NodeVoter,
					},
					State:  RaftVoter,
					Stats:  *serverStats["e72eb8da-604d-47cd-bd7f-69ec120ea2b7"],
					Health: ServerHealth{Healthy: true, StableSince: ts},
				},
			},
			Leader: "7875975d-d54b-49c1-a400-9fefcc706c67",
			Voters: []raft.ServerID{
				"7875975d-d54b-49c1-a400-9fefcc706c67",
				"e72eb8da-604d-47cd-bd7f-69ec120ea2b7",
				"ecfc5237-63c3-4b09-94b9-d5682d9ae5b1",
			},
		}
	}

	expected1 := genExpected(firstStateTime)
	expected2 := genExpected(restartStateTime)

	// these expectations are currently in the order that they are called in gatherNextStateInputs
	mdel.On("AutopilotConfig").Return(conf).Times(2)
	mraft.On("GetConfiguration").Return(&raftConfigFuture{config: test3VoterRaftConfiguration}).Times(2)
	mdel.On("KnownServers").Return(servers).Times(2)
	mraft.On("LastIndex").Return(lastIndex).Times(2)
	mraft.On("Stats").Return(map[string]string{"last_log_term": "3"}).Times(2)
	mdel.On("FetchServerStats", mock.Anything, servers).Return(serverStats).Times(2)
	mraft.On("Leader").Return(leaderAddr).Times(2)
	mdel.On("NotifyState", expected1).Once()
	mdel.On("NotifyState", expected2).Once()

	ap := New(mraft, mdel,
		WithReconcileInterval(30*time.Second),
		WithUpdateInterval(30*time.Second),
		withTimeProvider(mtime),
		WithLogger(testLogger(t)),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ap.Start(ctx)

	ap.execLock.Lock()
	require.NotNil(t, ap.execution)
	require.Equal(t, Running, ap.execution.status)
	require.NotNil(t, ap.execution.shutdown)
	require.NotNil(t, ap.execution.done)
	require.False(t, chanIsSelectable(ap.execution.done))
	ap.execLock.Unlock()

	status, ch := ap.IsRunning()
	require.Equal(t, Running, status)
	require.NotNil(t, ch)
	require.False(t, chanIsSelectable(ch))

	actual := ap.GetState()

	done := ap.Stop()
	require.NotNil(t, done)
	require.Eventually(t, func() bool {
		return chanIsSelectable(done)
	}, time.Second, 50*time.Millisecond)
	require.True(t, chanIsSelectable(ch))

	status, ch = ap.IsRunning()
	require.Equal(t, NotRunning, status)
	require.NotNil(t, ch)
	require.True(t, chanIsSelectable(ch))

	done = ap.Stop()
	require.NotNil(t, done)
	require.True(t, chanIsSelectable(done))
	require.Equal(t, expected1, actual)

	// restart things
	ap.Start(context.Background())
	// wait for the state to be generated
	require.Eventually(t, func() bool {
		return !ap.GetState().firstStateTime.IsZero()
	}, time.Second, 50*time.Millisecond)

	// get the currently running state
	actual = ap.GetState()
	// stop autopilot
	done = ap.Stop()
	// wait for it to actually be stopped
	require.NotNil(t, done)
	require.Eventually(t, func() bool {
		return chanIsSelectable(done)
	}, time.Second, 50*time.Millisecond)

	// ensure that the second state matches our expectations
	require.Equal(t, expected2, actual)

	// ensure that stopping caused the state to get erased
	require.NotNil(t, ap.state)
	require.Zero(t, *ap.state)

	// simulate shutting down of the previous go routine taking a long time
	ap.execution = &execInfo{
		status: ShuttingDown,
	}
	ap.leaderLock.Lock()

	// start autopilot while the execution lock is held. This
	// will cause the spawned go routine to sit idle until the
	// lock is relinquished or until it is cancelled.
	ap.Start(context.Background())
	require.NotNil(t, ap.execution)
	require.Equal(t, Running, ap.execution.status)
	// Note that because the pre-existing state was shuttingDown
	// then we are expecting no more calls to the various mocked
	// interfaces to ensure that this Start never gets to the
	// point of executing most of the code but instead gets
	// stuck in waiting on the lock and then cancelled.

	done = ap.Stop()
	require.NotNil(t, done)
	require.Eventually(t, func() bool {
		return chanIsSelectable(done)
	}, time.Second, 50*time.Millisecond)
}
