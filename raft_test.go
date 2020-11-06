package autopilot

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"
)

var (
	injectedErr = fmt.Errorf("injected err")

	test3VoterRaftConfiguration = raft.Configuration{
		Servers: []raft.Server{
			{
				Suffrage: raft.Voter,
				ID:       "7875975d-d54b-49c1-a400-9fefcc706c67",
				Address:  "198.18.0.1:8300",
			},
			{
				Suffrage: raft.Voter,
				ID:       "ecfc5237-63c3-4b09-94b9-d5682d9ae5b1",
				Address:  "198.18.0.2:8300",
			},
			{
				Suffrage: raft.Voter,
				ID:       "e72eb8da-604d-47cd-bd7f-69ec120ea2b7",
				Address:  "198.18.0.3:8300",
			},
		},
	}
)

func isInjectedError(err error) bool {
	return errors.Is(err, injectedErr)
}

func mockedRaftAutopilot(t *testing.T) (*Autopilot, *MockRaft) {
	t.Helper()
	mdel := NewMockApplicationIntegration(t)
	mraft := NewMockRaft(t)

	return New(mraft, mdel), mraft
}

func TestNumVoters(t *testing.T) {
	type testCase struct {
		future raftConfigFuture

		expected int
	}

	cases := map[string]testCase{
		"error": {
			future: raftConfigFuture{
				err: injectedErr,
			},
		},
		"all-voters": {
			future: raftConfigFuture{
				config: test3VoterRaftConfiguration,
			},
			expected: 3,
		},
		"ignore-staging": {
			future: raftConfigFuture{
				config: raft.Configuration{
					Servers: []raft.Server{
						{
							Suffrage: raft.Voter,
							ID:       "7875975d-d54b-49c1-a400-9fefcc706c67",
							Address:  "198.18.0.1:8300",
						},
						{
							Suffrage: raft.Staging,
							ID:       "ecfc5237-63c3-4b09-94b9-d5682d9ae5b1",
							Address:  "198.18.0.2:8300",
						},
						{
							Suffrage: raft.Voter,
							ID:       "e72eb8da-604d-47cd-bd7f-69ec120ea2b7",
							Address:  "198.18.0.3:8300",
						},
					},
				},
			},
			expected: 2,
		},
		"ignore-non-voter": {
			future: raftConfigFuture{
				config: raft.Configuration{
					Servers: []raft.Server{
						{
							Suffrage: raft.Voter,
							ID:       "7875975d-d54b-49c1-a400-9fefcc706c67",
							Address:  "198.18.0.1:8300",
						},
						{
							Suffrage: raft.Nonvoter,
							ID:       "ecfc5237-63c3-4b09-94b9-d5682d9ae5b1",
							Address:  "198.18.0.2:8300",
						},
						{
							Suffrage: raft.Voter,
							ID:       "e72eb8da-604d-47cd-bd7f-69ec120ea2b7",
							Address:  "198.18.0.3:8300",
						},
					},
				},
			},
			expected: 2,
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			ap, mraft := mockedRaftAutopilot(t)
			mraft.On("GetConfiguration").Return(&tcase.future).Once()

			voters, err := ap.NumVoters()
			if tcase.future.Error() != nil {
				require.Zero(t, voters)
				// error should just be passed through without modification
				require.Equal(t, tcase.future.Error(), err)
			} else {
				require.Equal(t, tcase.expected, voters)
				require.Nil(t, err)
			}
		})
	}
}

func TestAddServer(t *testing.T) {
	t.Run("existing-no-change", func(t *testing.T) {
		ap, mraft := mockedRaftAutopilot(t)

		mraft.On("GetConfiguration").Return(&raftConfigFuture{config: test3VoterRaftConfiguration}).Once()

		require.Nil(t, ap.AddServer(&Server{ID: "e72eb8da-604d-47cd-bd7f-69ec120ea2b7", Address: "198.18.0.3:8300"}))
		require.False(t, chanIsSelectable(ap.removeDeadCh))
	})

	t.Run("config-error", func(t *testing.T) {
		ap, mraft := mockedRaftAutopilot(t)

		mraft.On("GetConfiguration").Return(&raftConfigFuture{err: injectedErr}).Once()

		err := ap.AddServer(&Server{ID: "e72eb8da-604d-47cd-bd7f-69ec120ea2b7", Address: "198.18.0.3:8300"})
		require.Error(t, err)
		require.True(t, isInjectedError(err))
		require.False(t, chanIsSelectable(ap.removeDeadCh))
	})

	t.Run("new", func(t *testing.T) {
		ap, mraft := mockedRaftAutopilot(t)

		var newID raft.ServerID = "5e816fb6-d4e6-4b3a-b15a-afb3e6d5664b"
		var newAddr raft.ServerAddress = "198.18.0.4:8300"

		mraft.On("GetConfiguration").Return(&raftConfigFuture{config: test3VoterRaftConfiguration}).Once()
		mraft.On("AddNonvoter", newID, newAddr, uint64(0), time.Duration(0)).Return(&raftIndexFuture{}).Once()

		require.Nil(t, ap.AddServer(&Server{ID: newID, Address: newAddr}))
		require.True(t, chanIsSelectable(ap.removeDeadCh))
	})

	t.Run("existing-addr-change", func(t *testing.T) {
		ap, mraft := mockedRaftAutopilot(t)

		var existingID raft.ServerID = "ecfc5237-63c3-4b09-94b9-d5682d9ae5b1"
		var newAddr raft.ServerAddress = "198.18.0.4:8300"

		mraft.On("GetConfiguration").Return(&raftConfigFuture{config: test3VoterRaftConfiguration}).Once()
		mraft.On("AddVoter", existingID, newAddr, uint64(0), time.Duration(0)).Return(&raftIndexFuture{}).Once()

		require.Nil(t, ap.AddServer(&Server{ID: existingID, Address: newAddr}))
		require.True(t, chanIsSelectable(ap.removeDeadCh))
	})

	t.Run("existing-id-change", func(t *testing.T) {
		ap, mraft := mockedRaftAutopilot(t)

		var existingID raft.ServerID = "ecfc5237-63c3-4b09-94b9-d5682d9ae5b1"
		var newID raft.ServerID = "95e2f84d-ff36-4a48-bee7-a50863f17f55"
		var existingAddr raft.ServerAddress = "198.18.0.2:8300"

		mraft.On("GetConfiguration").Return(&raftConfigFuture{config: test3VoterRaftConfiguration}).Once()
		mraft.On("RemoveServer", existingID, uint64(0), time.Duration(0)).Return(&raftIndexFuture{}).Once()
		mraft.On("AddNonvoter", newID, existingAddr, uint64(0), time.Duration(0)).Return(&raftIndexFuture{}).Once()

		require.Nil(t, ap.AddServer(&Server{ID: newID, Address: existingAddr}))
		require.True(t, chanIsSelectable(ap.removeDeadCh))
	})

	t.Run("remove-failure", func(t *testing.T) {
		ap, mraft := mockedRaftAutopilot(t)

		var existingID raft.ServerID = "ecfc5237-63c3-4b09-94b9-d5682d9ae5b1"
		var newID raft.ServerID = "95e2f84d-ff36-4a48-bee7-a50863f17f55"
		var existingAddr raft.ServerAddress = "198.18.0.2:8300"

		mraft.On("GetConfiguration").Return(&raftConfigFuture{config: test3VoterRaftConfiguration}).Once()
		mraft.On("RemoveServer", existingID, uint64(0), time.Duration(0)).Return(&raftIndexFuture{err: injectedErr}).Once()

		require.True(t, isInjectedError(ap.AddServer(&Server{ID: newID, Address: existingAddr})))
		require.False(t, chanIsSelectable(ap.removeDeadCh))
	})

	t.Run("add-failure", func(t *testing.T) {
		ap, mraft := mockedRaftAutopilot(t)

		var newID raft.ServerID = "5e816fb6-d4e6-4b3a-b15a-afb3e6d5664b"
		var newAddr raft.ServerAddress = "198.18.0.4:8300"

		mraft.On("GetConfiguration").Return(&raftConfigFuture{config: test3VoterRaftConfiguration}).Once()
		mraft.On("AddNonvoter", newID, newAddr, uint64(0), time.Duration(0)).Return(&raftIndexFuture{err: injectedErr}).Once()

		require.True(t, isInjectedError(ap.AddServer(&Server{ID: newID, Address: newAddr})))
		require.False(t, chanIsSelectable(ap.removeDeadCh))
	})

	t.Run("error-for-unsafe-removals", func(t *testing.T) {
		ap, mraft := mockedRaftAutopilot(t)

		var newID raft.ServerID = "c3195ec1-c229-48c6-bdab-4db54ed04807"
		var newAddr raft.ServerAddress = "198.18.0.2:8300"

		mraft.On("GetConfiguration").Once().Return(&raftConfigFuture{
			config: raft.Configuration{
				Servers: []raft.Server{
					{
						ID:       "36ced65e-be2c-478d-a2f4-56a9706041d0",
						Address:  "198.18.0.1:8300",
						Suffrage: raft.Voter,
					},
					{
						ID:       "e0c54c7c-1363-46d0-950b-2cb4aad347a8",
						Address:  "198.18.0.2:8300",
						Suffrage: raft.Voter,
					},
				},
			},
		})

		err := ap.AddServer(&Server{ID: newID, Address: newAddr})
		require.Error(t, err)
		require.Contains(t, err.Error(), "Preventing server addition that would require removal of too many servers and cause cluster instability")
	})

	t.Run("safe-non-voter-removals", func(t *testing.T) {
		ap, mraft := mockedRaftAutopilot(t)

		var existingID raft.ServerID = "e0c54c7c-1363-46d0-950b-2cb4aad347a8"
		var newID raft.ServerID = "c3195ec1-c229-48c6-bdab-4db54ed04807"
		var newAddr raft.ServerAddress = "198.18.0.2:8300"

		mraft.On("GetConfiguration").Once().Return(&raftConfigFuture{
			config: raft.Configuration{
				Servers: []raft.Server{
					{
						ID:       "36ced65e-be2c-478d-a2f4-56a9706041d0",
						Address:  "198.18.0.1:8300",
						Suffrage: raft.Voter,
					},
					{
						ID:       existingID,
						Address:  "198.18.0.2:8300",
						Suffrage: raft.Nonvoter,
					},
				},
			},
		})

		mraft.On("RemoveServer", existingID, uint64(0), time.Duration(0)).Return(&raftIndexFuture{}).Once()
		mraft.On("AddNonvoter", newID, newAddr, uint64(0), time.Duration(0)).Return(&raftIndexFuture{}).Once()
		require.NoError(t, ap.AddServer(&Server{ID: newID, Address: newAddr}))
		require.True(t, chanIsSelectable(ap.removeDeadCh))
	})
}

func TestRemoveServer(t *testing.T) {
	t.Run("config-failure", func(t *testing.T) {
		ap, mraft := mockedRaftAutopilot(t)

		mraft.On("GetConfiguration").Return(&raftConfigFuture{err: injectedErr}).Once()
		require.True(t, isInjectedError(ap.RemoveServer("ecfc5237-63c3-4b09-94b9-d5682d9ae5b1")))
	})

	t.Run("not-found", func(t *testing.T) {
		ap, mraft := mockedRaftAutopilot(t)

		mraft.On("GetConfiguration").Return(&raftConfigFuture{config: test3VoterRaftConfiguration}).Once()
		require.NoError(t, ap.RemoveServer("29a3d904-6848-4e2f-928f-9abafc3f87ba"))
	})

	t.Run("removed", func(t *testing.T) {
		ap, mraft := mockedRaftAutopilot(t)

		var id raft.ServerID = "ecfc5237-63c3-4b09-94b9-d5682d9ae5b1"

		mraft.On("GetConfiguration").Return(&raftConfigFuture{config: test3VoterRaftConfiguration}).Once()
		mraft.On("RemoveServer", id, uint64(0), time.Duration(0)).Return(&raftIndexFuture{}).Once()
		require.NoError(t, ap.RemoveServer(id))
	})

	t.Run("remove-failure", func(t *testing.T) {
		ap, mraft := mockedRaftAutopilot(t)

		var id raft.ServerID = "ecfc5237-63c3-4b09-94b9-d5682d9ae5b1"

		mraft.On("GetConfiguration").Return(&raftConfigFuture{config: test3VoterRaftConfiguration}).Once()
		mraft.On("RemoveServer", id, uint64(0), time.Duration(0)).Return(&raftIndexFuture{err: injectedErr}).Once()
		require.True(t, isInjectedError(ap.RemoveServer(id)))
	})
}
