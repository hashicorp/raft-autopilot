package autopilot

import (
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
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
	conf := &Config{
		MaxTrailingLogs:      200,
		LastContactThreshold: 100 * time.Millisecond,
	}

	type testCase struct {
		server    ServerState
		expected  bool
		lastTerm  uint64
		lastIndex uint64
	}

	cases := map[string]testCase{
		"ok": {
			server: ServerState{
				Server: Server{NodeStatus: NodeAlive},
				Stats: ServerStats{
					LastContact: 99 * time.Millisecond,
					LastTerm:    5,
					LastIndex:   801,
				},
			},
			lastTerm:  5,
			lastIndex: 1000,
			expected:  true,
		},
		"node-failed": {
			server: ServerState{
				Server: Server{NodeStatus: NodeFailed},
				Stats: ServerStats{
					LastContact: 99 * time.Millisecond,
					LastTerm:    5,
					LastIndex:   801,
				},
			},
			lastTerm:  5,
			lastIndex: 1000,
			expected:  false,
		},
		"bad-raft-term": {
			server: ServerState{
				Server: Server{NodeStatus: NodeAlive},
				Stats: ServerStats{
					LastContact: 99 * time.Millisecond,
					LastTerm:    5 + 1,
					LastIndex:   801,
				},
			},
			lastTerm:  5,
			lastIndex: 1000,
			expected:  false,
		},
		"too-stale": {
			server: ServerState{
				Server: Server{NodeStatus: NodeAlive},
				Stats: ServerStats{
					LastContact: 150 * time.Millisecond,
					LastTerm:    5,
					LastIndex:   801,
				},
			},
			lastTerm:  5,
			lastIndex: 1000,
			expected:  false,
		},
		"index-too-old": {
			server: ServerState{
				Server: Server{NodeStatus: NodeAlive},
				Stats: ServerStats{
					LastContact: 99 * time.Millisecond,
					LastTerm:    5,
					LastIndex:   799,
				},
			},
			lastTerm:  5,
			lastIndex: 1000,
			expected:  false,
		},
		"no-leader": {
			server: ServerState{
				Server: Server{NodeStatus: NodeAlive},
				Stats: ServerStats{
					LastContact: 99 * time.Millisecond,
					LastTerm:    5,
					LastIndex:   801,
				},
			},
			lastTerm:  0,
			lastIndex: 0,
			expected:  false,
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tcase.expected, tcase.server.isHealthy(tcase.lastTerm, tcase.lastIndex, conf))
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

func TestFilterVoters(t *testing.T) {
	type testCase struct {
		servers     *RaftServers
		param       bool
		numExpected int
		expectedIds []string
	}

	cases := map[string]testCase{
		"no-servers": {
			servers:     &RaftServers{},
			param:       true,
			numExpected: 0,
		},
		"no-current-voters-when-true-single": {
			servers: &RaftServers{
				"abc123": &VoterEligibility{
					currentVoter:   false,
					potentialVoter: false,
				},
			},
			param:       true,
			numExpected: 0,
		},
		"current-voters-when-true-single": {
			servers: &RaftServers{
				"abc123": &VoterEligibility{
					currentVoter:   true,
					potentialVoter: true,
				},
			},
			param:       true,
			numExpected: 1,
			expectedIds: []string{"abc123"},
		},
		"no-current-voters-when-true-multiple": {
			servers: &RaftServers{
				"abc123": &VoterEligibility{
					currentVoter:   false,
					potentialVoter: false,
				},
				"456def": &VoterEligibility{
					currentVoter:   false,
					potentialVoter: true,
				},
			},
			param:       true,
			numExpected: 0,
		},
		"current-voters-when-true-multiple": {
			servers: &RaftServers{
				"abc123": &VoterEligibility{
					currentVoter:   true,
					potentialVoter: true,
				},
				"def456": &VoterEligibility{
					currentVoter:   true,
					potentialVoter: true,
				},
			},
			param:       true,
			numExpected: 2,
			expectedIds: []string{"abc123", "def456"},
		},
		"non-voters-when-false-single": {
			servers: &RaftServers{
				"abc123": &VoterEligibility{
					currentVoter:   false,
					potentialVoter: true,
				},
			},
			param:       false,
			numExpected: 1,
			expectedIds: []string{"abc123"},
		},
	}

	for name, tcase := range cases {
		name := name
		tcase := tcase
		t.Run(name, func(t *testing.T) {
			got := tcase.servers.FilterVoters(tcase.param)
			require.Equal(t, tcase.numExpected, len(got))
			for _, key := range tcase.expectedIds {
				assert.Contains(t, got, raft.ServerID(key))
			}
		})
	}
}

func TestGetPotentialVotersFromCategorizedServers(t *testing.T) {
	//func (s *CategorizedServers) PotentialVoters() int {
	type testCase struct {
		servers     *CategorizedServers
		numExpected int
	}

	cases := map[string]testCase{
		"no-servers": {
			servers:     &CategorizedServers{},
			numExpected: 0,
		},
		"single-one-of-each": {
			servers: &CategorizedServers{
				StaleNonVoters: RaftServers{
					"abc": &VoterEligibility{currentVoter: false, potentialVoter: false},
				},
				StaleVoters: RaftServers{
					"def": &VoterEligibility{currentVoter: true, potentialVoter: false},
				},
				FailedNonVoters: RaftServers{
					"ghi": &VoterEligibility{currentVoter: false, potentialVoter: true},
				},
				FailedVoters: RaftServers{
					"jkl": &VoterEligibility{currentVoter: false, potentialVoter: true},
				},
				HealthyNonVoters: RaftServers{
					"mno": &VoterEligibility{currentVoter: false, potentialVoter: true},
				},
				HealthyVoters: RaftServers{
					"pqr": &VoterEligibility{currentVoter: true, potentialVoter: true},
				},
			},
			numExpected: 4,
		},
	}

	for name, tcase := range cases {
		name := name
		tcase := tcase
		t.Run(name, func(t *testing.T) {
			got := tcase.servers.PotentialVoters()
			require.Equal(t, tcase.numExpected, got)
		})
	}
}
