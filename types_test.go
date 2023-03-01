package autopilot

import (
	"testing"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/raft"
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

func TestDistinct_Failed(t *testing.T) {
	type testCase struct {
		input    []*Server
		expected []*Server
	}

	cases := map[string]testCase{
		"no-servers": {
			input:    nil,
			expected: nil,
		},
		"single-server": {
			input: []*Server{
				{ID: "123"},
			},
			expected: []*Server{
				{ID: "123"},
			},
		},
		"multi-server-unique": {
			input: []*Server{
				{ID: "123"},
				{ID: "456"},
				{ID: "789"},
			},
			expected: []*Server{
				{ID: "123"},
				{ID: "456"},
				{ID: "789"},
			},
		},
		"multi-server-dupes": {
			input: []*Server{
				{ID: "123"},
				{ID: "123"},
				{ID: "789"},
			},
			expected: []*Server{
				{ID: "123"},
				{ID: "789"},
			},
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, len(tcase.expected), len(distinctFailed(tcase.input)))
			require.Equal(t, tcase.expected, distinctFailed(tcase.input))
		})
	}
}

func TestDistinct_Stale(t *testing.T) {
	type testCase struct {
		input    []raft.ServerID
		expected []raft.ServerID
	}

	cases := map[string]testCase{
		"no-servers": {
			input:    nil,
			expected: nil,
		},
		"single-server": {
			input:    []raft.ServerID{"123"},
			expected: []raft.ServerID{"123"},
		},
		"multi-server-unique": {
			input:    []raft.ServerID{"123", "456", "789"},
			expected: []raft.ServerID{"123", "456", "789"},
		},
		"multi-server-dupes": {
			input:    []raft.ServerID{"123", "123", "789"},
			expected: []raft.ServerID{"123", "789"},
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, len(tcase.expected), len(distinctStale(tcase.input)))
			require.Equal(t, tcase.expected, distinctStale(tcase.input))
		})
	}
}

func TestDistinct_FailedServers(t *testing.T) {
	type testCase struct {
		input               FailedServers
		failedVoterCount    int
		failedNonVoterCount int
		staleVoterCount     int
		staleNonVoterCount  int
	}

	cases := map[string]testCase{
		"no-servers": {
			input: FailedServers{
				FailedVoters:    nil,
				FailedNonVoters: nil,
				StaleVoters:     nil,
				StaleNonVoters:  nil,
			},
			failedVoterCount:    0,
			failedNonVoterCount: 0,
			staleVoterCount:     0,
			staleNonVoterCount:  0,
		},
		"single-server-per-category": {
			input: FailedServers{
				FailedVoters: []*Server{
					{ID: "123"},
				},
				FailedNonVoters: []*Server{
					{ID: "456"},
				},
				StaleVoters: []raft.ServerID{
					"789",
				},
				StaleNonVoters: []raft.ServerID{
					"000",
				},
			},
			failedVoterCount:    1,
			failedNonVoterCount: 1,
			staleVoterCount:     1,
			staleNonVoterCount:  1,
		},
		"multi-server-unique-per-category": {
			input: FailedServers{
				FailedVoters: []*Server{
					{ID: "123"},
					{ID: "456"},
					{ID: "789"},
				},
				FailedNonVoters: []*Server{
					{ID: "abc"},
					{ID: "def"},
					{ID: "ghi"},
				},
				StaleVoters: []raft.ServerID{
					"xxx", "yyy",
				},
				StaleNonVoters: []raft.ServerID{
					"zzz", "000",
				},
			},
			failedVoterCount:    3,
			failedNonVoterCount: 3,
			staleVoterCount:     2,
			staleNonVoterCount:  2,
		},
		"multi-server-dupes-per-category": {
			input: FailedServers{
				FailedVoters: []*Server{
					{ID: "123"},
					{ID: "123"},
					{ID: "789"},
				},
				FailedNonVoters: []*Server{
					{ID: "abc"},
					{ID: "def"},
					{ID: "def"},
				},
				StaleVoters: []raft.ServerID{
					"xxx", "xxx",
				},
				StaleNonVoters: []raft.ServerID{
					"zzz", "000", "zzz", "juan",
				},
			},
			failedVoterCount:    2,
			failedNonVoterCount: 2,
			staleVoterCount:     1,
			staleNonVoterCount:  3,
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			tcase.input.distinct()

			require.Equal(t, tcase.failedVoterCount, len(tcase.input.FailedVoters))
			require.Equal(t, tcase.failedNonVoterCount, len(tcase.input.FailedNonVoters))
			require.Equal(t, tcase.staleVoterCount, len(tcase.input.StaleVoters))
			require.Equal(t, tcase.staleNonVoterCount, len(tcase.input.StaleNonVoters))
		})
	}
}

func TestExclusive_Failed(t *testing.T) {
	type testCase struct {
		inputServers []*Server
		inputKeys    map[raft.ServerID]string
		inputLabel   string
		expected     string
	}

	cases := map[string]testCase{
		"no-servers": {
			inputServers: nil,
			inputKeys:    nil,
			inputLabel:   "failed-new",
			expected:     "",
		},
		"single-server": {
			inputServers: []*Server{
				{ID: "123"},
			},
			inputKeys: map[raft.ServerID]string{
				raft.ServerID("abc"): "failed-old",
			},
			inputLabel: "failed-new",
			expected:   "",
		},
		"multiple-servers-unique-not-seen": {
			inputServers: []*Server{
				{ID: "123"},
				{ID: "456"},
			},
			inputKeys: map[raft.ServerID]string{
				raft.ServerID("abc"): "failed-old",
			},
			inputLabel: "failed-new",
			expected:   "",
		},
		"multiple-servers-unique-seen": {
			inputServers: []*Server{
				{ID: "123"},
				{ID: "456"},
			},
			inputKeys: map[raft.ServerID]string{
				raft.ServerID("123"): "failed-old",
			},
			inputLabel: "failed-new",
			expected:   "parsing \"failed-new\", duplicate ID: 123 found in \"failed-old\"",
		},
		"multiple-servers-dupes-not-seen": {
			inputServers: []*Server{
				{ID: "123"},
				{ID: "123"},
			},
			inputKeys: map[raft.ServerID]string{
				raft.ServerID("abc"): "failed-old",
			},
			inputLabel: "failed-new",
			expected:   "parsing \"failed-new\", duplicate ID: 123 found in \"failed-new\"",
		},
		"multiple-servers-dupes-seen": {
			inputServers: []*Server{
				{ID: "123"},
				{ID: "123"},
			},
			inputKeys: map[raft.ServerID]string{
				raft.ServerID("123"): "failed-old",
			},
			inputLabel: "failed-new",
			expected:   "parsing \"failed-new\", duplicate ID: 123 found in \"failed-old\"",
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := exclusiveFailed(tcase.inputServers, tcase.inputKeys, tcase.inputLabel)

			if tcase.expected == "" && err != nil {
				t.Fatal("Error was not expected", err)
			} else if merr, ok := err.(*multierror.Error); ok {
				// Use merr.Errors
				for _, werr := range merr.WrappedErrors() {
					require.Equal(t, tcase.expected, werr.Error())
				}
			}
		})
	}
}

func TestExclusive_Stale(t *testing.T) {
	type testCase struct {
		inputServers []raft.ServerID
		inputKeys    map[raft.ServerID]string
		inputLabel   string
		expected     []string
	}

	cases := map[string]testCase{
		"no-servers": {
			inputServers: nil,
			inputKeys:    nil,
			inputLabel:   "failed-new",
			expected:     nil,
		},
		"single-server": {
			inputServers: []raft.ServerID{"123"},
			inputKeys: map[raft.ServerID]string{
				raft.ServerID("abc"): "failed-old",
			},
			inputLabel: "failed-new",
			expected:   nil,
		},
		"multiple-servers-unique-not-seen": {
			inputServers: []raft.ServerID{"123", "456"},
			inputKeys: map[raft.ServerID]string{
				raft.ServerID("abc"): "failed-old",
			},
			inputLabel: "failed-new",
			expected:   nil,
		},
		"multiple-servers-unique-seen": {
			inputServers: []raft.ServerID{"123", "456"},
			inputKeys: map[raft.ServerID]string{
				raft.ServerID("123"): "failed-old",
			},
			inputLabel: "failed-new",
			expected: []string{
				"parsing \"failed-new\", duplicate ID: 123 found in \"failed-old\"",
			},
		},
		"multiple-servers-dupes-not-seen": {
			inputServers: []raft.ServerID{"123", "123"},
			inputKeys: map[raft.ServerID]string{
				raft.ServerID("abc"): "failed-old",
			},
			inputLabel: "failed-new",
			expected: []string{
				"parsing \"failed-new\", duplicate ID: 123 found in \"failed-new\"",
			},
		},
		"multiple-servers-dupe-seen": {
			inputServers: []raft.ServerID{"123", "123"},
			inputKeys: map[raft.ServerID]string{
				raft.ServerID("123"): "failed-old",
			},
			inputLabel: "failed-new",
			expected: []string{
				"parsing \"failed-new\", duplicate ID: 123 found in \"failed-old\"",
				"parsing \"failed-new\", duplicate ID: 123 found in \"failed-old\"",
			},
		},
		"multiple-servers-dupes-seen": {
			inputServers: []raft.ServerID{"123", "123", "456", "456"},
			inputKeys: map[raft.ServerID]string{
				raft.ServerID("123"): "failed-old",
				raft.ServerID("456"): "failed-old",
			},
			inputLabel: "failed-new",
			expected: []string{
				"parsing \"failed-new\", duplicate ID: 123 found in \"failed-old\"",
				"parsing \"failed-new\", duplicate ID: 123 found in \"failed-old\"",
				"parsing \"failed-new\", duplicate ID: 456 found in \"failed-old\"",
				"parsing \"failed-new\", duplicate ID: 456 found in \"failed-old\"",
			},
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := exclusiveStale(tcase.inputServers, tcase.inputKeys, tcase.inputLabel)

			if tcase.expected == nil && err != nil {
				t.Fatal("Error was not expected", err)
			} else if merr, ok := err.(*multierror.Error); ok {
				for _, werr := range merr.WrappedErrors() {
					found := false
					for _, v := range tcase.expected {
						if v == werr.Error() {
							found = true
						}
					}
					require.True(t, found, "missing error: "+werr.Error())
				}
				require.Equal(t, len(tcase.expected), merr.Len())
			}
		})
	}
}
