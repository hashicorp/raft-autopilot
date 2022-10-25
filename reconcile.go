package autopilot

import (
	"fmt"

	"github.com/hashicorp/raft"
)

// reconcile calculates and then applies promotions and demotions
func (a *Autopilot) reconcile() error {
	if !a.ReconciliationEnabled() {
		return nil
	}

	conf := a.delegate.AutopilotConfig()
	if conf == nil {
		return nil
	}

	// grab the current state while locked
	state := a.GetState()

	if state == nil || state.Leader == "" {
		return fmt.Errorf("cannot reconcile Raft server voting rights without a valid autopilot state")
	}

	// have the promoter calculate the required Raft changeset.
	changes := a.promoter.CalculatePromotionsAndDemotions(conf, state)

	// apply the promotions, if we did apply any then stop here
	// as we do not want to apply the demotions at the same time
	// as a means of preventing cluster instability.
	if done, err := a.applyPromotions(state, changes); done {
		return err
	}

	// apply the demotions, if we did apply any then stop here
	// as we do not want to transition leadership and do demotions
	// at the same time. This is a preventative measure to maintain
	// cluster stability.
	if done, err := a.applyDemotions(state, changes); done {
		return err
	}

	// if no leadership transfer is desired then we can exit the method now.
	if changes.Leader == "" || changes.Leader == state.Leader {
		return nil
	}

	// lookup the server we want to transfer leadership to
	srv, ok := state.Servers[changes.Leader]
	if !ok {
		return fmt.Errorf("cannot transfer leadership to an unknown server with ID %s", changes.Leader)
	}

	// perform the leadership transfer
	return a.leadershipTransfer(changes.Leader, srv.Server.Address)
}

// applyPromotions will apply all the promotions in the RaftChanges parameter.
//
// IDs in the change set will be ignored if:
// * The server isn't tracked in the provided state
// * The server already has voting rights
// * The server is not healthy
//
// If any servers were promoted this function returns true for the bool value.
func (a *Autopilot) applyPromotions(state *State, changes RaftChanges) (bool, error) {
	promoted := false
	for _, change := range changes.Promotions {
		srv, found := state.Servers[change]
		if !found {
			a.logger.Debug("Ignoring promotion of server as it is not in the autopilot state", "id", change)
			// this shouldn't be able to happen but is a nice safety measure against the
			// delegate doing something less than desirable
			continue
		}

		if srv.HasVotingRights() {
			// There is no need to promote as this server is already a voter.
			// No logging is needed here as this could be a very common case
			// where the promoter just returns a lists of server ids that should
			// be voters and non-voters without caring about which ones currently
			// already are in that state.
			a.logger.Debug("Not promoting server that already has voting rights", "id", change)
			continue
		}

		if !srv.Health.Healthy {
			// do not promote unhealthy servers
			a.logger.Debug("Ignoring promotion of unhealthy server", "id", change)
			continue
		}

		a.logger.Info("Promoting server", "id", srv.Server.ID, "address", srv.Server.Address, "name", srv.Server.Name)

		if err := a.addVoter(srv.Server.ID, srv.Server.Address); err != nil {
			return true, fmt.Errorf("failed promoting server %s: %v", srv.Server.ID, err)
		}

		promoted = true
	}

	// when we promoted anything we return true to indicate that the promotion/demotion applying
	// process is finished to prevent promotions and demotions in the same round. This is what
	// autopilot within Consul used to do so I am keeping the behavior the same for now.
	return promoted, nil
}

// applyDemotions will apply all the demotions in the RaftChanges parameter.
//
// IDs in the change set will be ignored if:
// * The server isn't tracked in the provided state
// * The server does not have voting rights
//
// If any servers were demoted this function returns true for the bool value.
func (a *Autopilot) applyDemotions(state *State, changes RaftChanges) (bool, error) {
	demoted := false
	for _, change := range changes.Demotions {
		srv, found := state.Servers[change]
		if !found {
			a.logger.Debug("Ignoring demotion of server as it is not in the autopilot state", "id", change)
			// this shouldn't be able to happen but is a nice safety measure against the
			// delegate doing something less than desirable
			continue
		}

		if srv.State == RaftNonVoter {
			// There is no need to demote as this server is already a non-voter.
			// No logging is needed here as this could be a very common case
			// where the promoter just returns a lists of server ids that should
			// be voters and non-voters without caring about which ones currently
			// already are in that state.
			a.logger.Debug("Ignoring demotion of server that is already a non-voter", "id", change)
			continue
		}

		a.logger.Info("Demoting server", "id", srv.Server.ID, "address", srv.Server.Address, "name", srv.Server.Name)

		if err := a.demoteVoter(srv.Server.ID); err != nil {
			return true, fmt.Errorf("failed demoting server %s: %v", srv.Server.ID, err)
		}

		demoted = true
	}

	// similarly to applyPromotions here we want to stop the process and prevent leadership
	// transfer when any demotions took place. Basically we want to ensure the cluster is
	// stable before doing the transfer
	return demoted, nil
}

func (a *Autopilot) categorizeServers(voterPredicate func(NodeType) bool) (*CategorizedServers, error) {
	cfg, err := a.getRaftConfiguration()
	if err != nil {
		return nil, err
	}

	// Get servers as raft sees them currently
	// (we won't know if they have the potential to become voters yet)
	raftServers := getServerSuffrage(cfg.Servers)
	failedVoters := make(RaftServerEligibility)
	failedNonVoters := make(RaftServerEligibility)
	healthyVoters := make(RaftServerEligibility)
	healthyNonVoters := make(RaftServerEligibility)

	// Loop over all the servers the application knows about
	for id, srv := range a.delegate.KnownServers() {
		v, found := raftServers[id]
		if !found {
			// This server was known to the application,
			// but not in the Raft config, so will be ignored
			continue
		}

		delete(raftServers, id)

		if srv.NodeStatus == NodeAlive && v.IsCurrentVoter() {
			healthyVoters[id] = v
		} else if srv.NodeStatus == NodeAlive {
			healthyNonVoters[id] = v
		} else if v.IsCurrentVoter() {
			failedVoters[id] = v
		} else {
			failedNonVoters[id] = v
		}

		v.SetPotentialVoter(voterPredicate(srv.NodeType))
	}

	c := &CategorizedServers{
		StaleNonVoters:   raftServers.FilterVoters(false),
		StaleVoters:      raftServers.FilterVoters(true),
		FailedNonVoters:  failedNonVoters,
		FailedVoters:     failedVoters,
		HealthyNonVoters: healthyNonVoters,
		HealthyVoters:    healthyVoters,
	}

	return c, nil
}

func (a *Autopilot) RemoveStaleServer(id raft.ServerID) error {
	a.logger.Debug("removing server by ID", "id", id)
	future := a.raft.RemoveServer(id, 0, 0)
	if err := future.Error(); err != nil {
		a.logger.Error("failed to remove raft server",
			"id", id,
			"error", err,
		)
		return err
	}
	a.logger.Info("removed server", "id", id)
	return nil
}

func (a *Autopilot) RemoveStaleServers(toRemove []raft.ServerID) error {
	for _, id := range toRemove {
		err := a.RemoveStaleServer(id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *Autopilot) RemoveFailedServer(id raft.ServerID) {
	srv, found := a.delegate.KnownServers()[id]
	if found {
		a.delegate.RemoveFailedServer(srv)
	}
}

func (a *Autopilot) RemoveFailedServers(toRemove []raft.ServerID) {
	for _, id := range toRemove {
		a.RemoveFailedServer(id)
	}
}

func (a *Autopilot) pruneDeadServers() error {
	if !a.ReconciliationEnabled() {
		return nil
	}

	conf := a.delegate.AutopilotConfig()
	if conf == nil || !conf.CleanupDeadServers {
		return nil
	}

	servers, err := a.categorizeServers(a.promoter.PotentialVoterPredicate)
	if err != nil {
		return err
	}

	state := a.GetState()

	// Support not breaking the promoter's interface for filtering servers
	failedServers := servers.convertToFailedServers(state)
	failedServers = a.promoter.FilterFailedServerRemovals(conf, state, failedServers)
	servers = servers.convertFromFailedServers(failedServers)

	// Curry adjudicate function
	adjudicate := func(voterCountProvider func() int) func(RaftServerEligibility) []raft.ServerID {
		return func(raftServers RaftServerEligibility) []raft.ServerID {
			return a.adjudicateRemoval(voterCountProvider, raftServers)
		}
	}(servers.PotentialVoters)

	// Try to remove servers in order of increasing precedence

	// Remove all stale non-voters
	if err = a.RemoveStaleServers(adjudicate(servers.StaleNonVoters)); err != nil {
		return err
	}

	// Remove stale voters
	if err = a.RemoveStaleServers(adjudicate(servers.StaleVoters)); err != nil {
		return err
	}

	// Remove failed non-voters
	a.RemoveFailedServers(adjudicate(servers.FailedNonVoters))

	// Remove failed voters
	a.RemoveFailedServers(adjudicate(servers.FailedVoters))

	return nil
}

func (a *Autopilot) adjudicateRemoval(voterCountProvider func() int, s RaftServerEligibility) []raft.ServerID {
	var ids []raft.ServerID
	failureTolerance := (voterCountProvider() - 1) / 2
	minQuorum := a.delegate.AutopilotConfig().MinQuorum

	for id, v := range s {
		if v != nil && v.IsPotentialVoter() && voterCountProvider()-1 < int(minQuorum) {
			a.logger.Debug("will not remove server node as it would leave less voters than the minimum number allowed", "id", id, "min", minQuorum)
		} else if failureTolerance < 1 {
			a.logger.Debug("will not remove server node as removal of a majority of servers is not safe", "id", id)
		} else if v != nil && v.IsCurrentVoter() {
			failureTolerance--
			delete(s, id)
			ids = append(ids, id)
		} else {
			delete(s, id)
			ids = append(ids, id)
		}
	}

	return ids
}

func getServerSuffrage(servers []raft.Server) RaftServerEligibility {
	ids := make(RaftServerEligibility)

	for _, server := range servers {
		ids[server.ID] = &VoterEligibility{
			currentVoter: server.Suffrage == raft.Voter,
		}
	}

	return ids
}
