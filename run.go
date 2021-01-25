package autopilot

import (
	"context"
	"time"
)

// Start will launch the go routines in the background to perform Autopilot.
// When the context passed in is cancelled or the Stop method is called
// then these routines will exit.
func (a *Autopilot) Start(ctx context.Context) error {
	a.runLock.Lock()
	defer a.runLock.Unlock()

	// already running so there is nothing to do
	if a.execution != nil && a.execution.status == Running {
		return nil
	}

	ctx, shutdown := context.WithCancel(ctx)
	a.startTime = a.time.Now()

	exec := &execInfo{
		status:   Running,
		shutdown: shutdown,
		done:     make(chan struct{}),
	}

	if a.execution == nil || a.execution.status == NotRunning {
		// While a go routine executed by a.run below will periodically
		// update the state, we want to go ahead and force updating it now
		// so that during a leadership transfer we don't report an empty
		// autopilot state. We put a pretty small timeout on this though
		// so as to prevent leader establishment from taking too long. This
		// only is done if we are not running and not shutting down so
		// as to prevent conflicts with any autopilot routine just finishing
		// up such as when restarting autopilot.
		updateCtx, updateCancel := context.WithTimeout(ctx, time.Second)
		defer updateCancel()
		a.updateState(updateCtx)
	}

	go a.run(ctx, exec)
	a.execution = exec
	return nil
}

// Stop will terminate the go routines being executed to perform autopilot.
func (a *Autopilot) Stop() <-chan struct{} {
	a.runLock.Lock()
	defer a.runLock.Unlock()

	// Nothing to do
	if a.execution == nil || a.execution.status == NotRunning {
		done := make(chan struct{})
		close(done)
		return done
	}

	a.execution.shutdown()
	a.execution.status = ShuttingDown
	return a.execution.done
}

// IsRunning returns the current execution status of the autopilot
// go routines as well as a chan which will be closed when the
// routines are no longer running
func (a *Autopilot) IsRunning() (ExecutionStatus, <-chan struct{}) {
	a.runLock.Lock()
	defer a.runLock.Unlock()

	if a.execution == nil || a.execution.status == NotRunning {
		done := make(chan struct{})
		close(done)
		return NotRunning, done
	}

	return a.execution.status, a.execution.done
}

func (a *Autopilot) endRun(exec *execInfo) {
	// need to gain the lock because if this was the active execution
	// then these values may be read while they are updated.
	a.runLock.Lock()
	defer a.runLock.Unlock()

	exec.shutdown = nil
	exec.status = NotRunning
	// this should be the final cleanup task as it is what notifies the rest
	// of the world that we are now done
	close(exec.done)
	exec.done = nil
}

func (a *Autopilot) run(ctx context.Context, exec *execInfo) {
	// This will wait for any other go routine to finish executing
	// before running any code ourselves to prevent any conflicting
	// activity between the two.
	if err := a.execMutex.TryLock(ctx); err != nil {
		a.endRun(exec)
		return
	}

	a.logger.Debug("autopilot is now running")

	// autopilot needs to do 3 things
	//
	// 1. periodically update the cluster state
	// 2. periodically check for and perform promotions and demotions
	// 3. Respond to servers leaving and prune dead servers
	//
	// We could attempt to do all of this in a single go routine except that
	// updating the cluster health could potentially take long enough to impact
	// the periodicity of the promotions and demotions performed by task 2/3.
	// So instead this go routine will spawn a second go routine to manage
	// updating the cluster health in the background. This go routine is still
	// in control of the overall running status and will not exit until the
	// child go routine has exited.

	// child go routine for cluster health updating
	stateUpdaterDone := make(chan struct{})
	go a.runStateUpdater(ctx, stateUpdaterDone)

	// cleanup for once we are stopped
	defer func() {
		// block waiting for our child go routine to also finish
		<-stateUpdaterDone

		a.logger.Debug("autopilot is now stopped")

		a.endRun(exec)
		a.execMutex.Unlock()
	}()

	reconcileTicker := time.NewTicker(a.reconcileInterval)
	defer reconcileTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-reconcileTicker.C:
			if err := a.reconcile(); err != nil {
				a.logger.Error("Failed to reconcile current state with the desired state")
			}

			if err := a.pruneDeadServers(); err != nil {
				a.logger.Error("Failed to prune dead servers", "error", err)
			}
		case <-a.removeDeadCh:
			if err := a.pruneDeadServers(); err != nil {
				a.logger.Error("Failed to prune dead servers", "error", err)
			}
		}
	}
}

// runStateUpdated will periodically update the autopilot state until the context
// passed in is cancelled. When finished the provide done chan will be closed.
func (a *Autopilot) runStateUpdater(ctx context.Context, done chan struct{}) {
	a.logger.Debug("state update routine is now running")
	defer func() {
		a.logger.Debug("state update routine is now stopped")
		close(done)
	}()

	ticker := time.NewTicker(a.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.updateState(ctx)
		}
	}
}
