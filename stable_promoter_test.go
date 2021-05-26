package autopilot

import (
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"
)

func TestStablePromoter_GetNodeTypes(t *testing.T) {
	state := &State{
		Servers: map[raft.ServerID]*ServerState{
			"462fca30-0947-4d5c-82e0-c549b0bf5b6d": {},
			"11a62e75-5418-481e-90eb-c238d796dca9": {},
			"f536ec02-f859-4e61-a484-c1e6a085ce46": {},
		},
	}

	expected := map[raft.ServerID]NodeType{
		"462fca30-0947-4d5c-82e0-c549b0bf5b6d": NodeVoter,
		"11a62e75-5418-481e-90eb-c238d796dca9": NodeVoter,
		"f536ec02-f859-4e61-a484-c1e6a085ce46": NodeVoter,
	}

	var promoter StablePromoter

	require.Equal(t, expected, promoter.GetNodeTypes(nil, state))
}

func TestStablePromoter_CalculatePromotionsAndDemotions(t *testing.T) {
	state := &State{
		firstStateTime: time.Now().Add(-30 * time.Second),
		Servers: map[raft.ServerID]*ServerState{
			// alread the leader - will not promote
			"462fca30-0947-4d5c-82e0-c549b0bf5b6d": {
				State: RaftLeader,
				Health: ServerHealth{
					Healthy: true,
				},
			},
			// already a voter - will not promote
			"11a62e75-5418-481e-90eb-c238d796dca9": {
				State: RaftVoter,
				Health: ServerHealth{
					Healthy:     true,
					StableSince: time.Now().Add(-20 * time.Second),
				},
			},
			// healthy stable voter - will promote
			"f536ec02-f859-4e61-a484-c1e6a085ce46": {
				State: RaftNonVoter,
				Health: ServerHealth{
					Healthy:     true,
					StableSince: time.Now().Add(-11 * time.Second),
				},
			},
			// unhealthy - will not promote
			"f94f3090-cd4c-4bca-9e24-97fb0535b3a4": {
				State: RaftNonVoter,
				Health: ServerHealth{
					Healthy:     false,
					StableSince: time.Now().Add(-11 * time.Second),
				},
			},
			// still staging - will not promote
			"ef7cecc1-4a49-491a-813a-fc666a22c3bc": {
				State: RaftStaging,
				Health: ServerHealth{
					Healthy:     true,
					StableSince: time.Now().Add(-11 * time.Second),
				},
			},
			// not stable long enough - will not promote
			"2d601ea3-3b51-4b8e-86da-aae5712c99e2": {
				State: RaftNonVoter,
				Health: ServerHealth{
					Healthy:     true,
					StableSince: time.Now().Add(-2 * time.Second),
				},
			},
		},
	}

	expected := RaftChanges{
		Promotions: []raft.ServerID{"f536ec02-f859-4e61-a484-c1e6a085ce46"},
	}

	var promoter StablePromoter
	conf := &Config{ServerStabilizationTime: 10 * time.Second}
	require.Equal(t, expected, promoter.CalculatePromotionsAndDemotions(conf, state))
}
