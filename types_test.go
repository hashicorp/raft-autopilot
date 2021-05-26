package autopilot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHasVotingRights(t *testing.T) {
	type testCase struct {
		state    RaftState
		expected bool
	}
	cases := map[string]testCase{
		"leader": {
			state:    RaftLeader,
			expected: true,
		},
		"voter": {
			state:    RaftVoter,
			expected: true,
		},
		"non-voter": {
			state:    RaftNonVoter,
			expected: false,
		},
		"staging": {
			state:    RaftStaging,
			expected: false,
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			state := ServerState{
				State: tcase.state,
			}

			require.Equal(t, tcase.expected, state.HasVotingRights())
		})
	}
}

func TestServerIsHealthy(t *testing.T) {
	// these are the settings that are going to be passed into the
	// isHealthy calls so the testCases should take them into account
	// for whether the overall expected health should be true/false
	var lastTerm uint64 = 5
	var leaderLastIndex uint64 = 1000
	conf := &Config{
		MaxTrailingLogs:      200,
		LastContactThreshold: 100 * time.Millisecond,
	}

	type testCase struct {
		server   ServerState
		expected bool
	}

	cases := map[string]testCase{
		"ok": {
			server: ServerState{
				Server: Server{NodeStatus: NodeAlive},
				Stats: ServerStats{
					LastContact: 99 * time.Millisecond,
					LastTerm:    lastTerm,
					LastIndex:   801,
				},
			},
			expected: true,
		},
		"node-failed": {
			server: ServerState{
				Server: Server{NodeStatus: NodeFailed},
				Stats: ServerStats{
					LastContact: 99 * time.Millisecond,
					LastTerm:    lastTerm,
					LastIndex:   801,
				},
			},
			expected: false,
		},
		"bad-raft-term": {
			server: ServerState{
				Server: Server{NodeStatus: NodeAlive},
				Stats: ServerStats{
					LastContact: 99 * time.Millisecond,
					LastTerm:    lastTerm + 1,
					LastIndex:   801,
				},
			},
			expected: false,
		},
		"too-stale": {
			server: ServerState{
				Server: Server{NodeStatus: NodeAlive},
				Stats: ServerStats{
					LastContact: 150 * time.Millisecond,
					LastTerm:    lastTerm,
					LastIndex:   801,
				},
			},
			expected: false,
		},
		"index-too-old": {
			server: ServerState{
				Server: Server{NodeStatus: NodeAlive},
				Stats: ServerStats{
					LastContact: 99 * time.Millisecond,
					LastTerm:    lastTerm,
					LastIndex:   799,
				},
			},
			expected: false,
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tcase.expected, tcase.server.isHealthy(lastTerm, leaderLastIndex, conf))
		})
	}
}

func TestServerIsStable(t *testing.T) {
	type testCase struct {
		health            *ServerHealth
		now               time.Time
		minStableDuration time.Duration
		expected          bool
	}

	cases := map[string]testCase{
		"nil": {
			expected: false,
		},
		"raft-unhealthy": {
			health: &ServerHealth{
				Healthy:     false,
				StableSince: time.Date(2020, 11, 2, 0, 0, 0, 0, time.UTC),
			},
			now:               time.Date(2020, 11, 2, 1, 0, 0, 0, time.UTC),
			minStableDuration: 10 * time.Second,
			expected:          false,
		},
		"not-stable": {
			health: &ServerHealth{
				Healthy:     true,
				StableSince: time.Date(2020, 11, 2, 1, 0, 0, 0, time.UTC),
			},
			now:               time.Date(2020, 11, 2, 1, 0, 1, 0, time.UTC),
			minStableDuration: 10 * time.Second,
			expected:          false,
		},
		"ok": {
			health: &ServerHealth{
				Healthy:     true,
				StableSince: time.Date(2020, 11, 2, 1, 0, 0, 0, time.UTC),
			},
			now:               time.Date(2020, 11, 2, 1, 0, 10, 0, time.UTC),
			minStableDuration: 10 * time.Second,
			expected:          true,
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tcase.expected, tcase.health.IsStable(tcase.now, tcase.minStableDuration))
		})
	}
}

func TestServerStabilizationTime(t *testing.T) {
	type testCase struct {
		serverStabilizationTime time.Duration
		expected                time.Duration
	}

	conf := &Config{
		ServerStabilizationTime: 350 * time.Millisecond,
	}

	s := &State{
		firstStateTime: time.Now(),
	}

	require.Equal(t, 0*time.Nanosecond, s.ServerStabilizationTime(conf))

	require.Eventually(t, func() bool {
		return s.ServerStabilizationTime(conf) == 350*time.Millisecond
	}, 500*time.Millisecond, 50*time.Millisecond)

}
