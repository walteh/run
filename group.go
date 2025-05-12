package run

import (
	"context"
	"time"
)

// group collects actors (functions) and runs them concurrently.
// When one actor (function) returns, all actors are interrupted.
// The zero value of a Group is useful.
type group struct {
	actors       []actor
	closeTimeout time.Duration
	syncShutdown bool
	deps map[ID][]ID
}


// Add an actor (function) to the group. Each actor must be pre-emptable by an
// interrupt function. That is, if interrupt is invoked, execute should return.
// Also, it must be safe to call interrupt even after execute has returned.
//
// The first actor (function) to return interrupts all running actors.
// The error is passed to the interrupt functions, and is returned by Run.
func (g *group) add(execute func(context.Context) error, interrupt func(context.Context) error, ready func() bool) ID {
	actor := actor{execute, interrupt, ready, NewID()}
	g.actors = append(g.actors, actor)
	return actor.id
}

func (g *group) findByName(name ID) (*actor, bool) {
	for _, a := range g.actors {
		if a.id == name {
			return &a, true
		}
	}
	return nil, false
}

// Run all actors (functions) concurrently.
// When the first actor returns, all others are interrupted.
// Run only returns when all actors have exited.
// Run returns the error returned by the first exiting actor.
func (g *group) run() error {
	return g.runContext(context.Background())
}

func (g *group) runContext(inputCtx context.Context) error {
	if len(g.actors) == 0 {
		return nil
	}


	ready := make(chan actor, len(g.actors))
	defer close(ready)

	for _, a := range g.actors {
		myDeps := g.deps[a.id]
		go func(a actor) {
			for {
				ready := true
				for _, dep := range myDeps {
					depActor, ok := g.findByName(dep)
					if !ok {
						panic("dependency not found: " + dep)
					}
					if !depActor.ready() {
						ready = false
						break
					}
				}
				if ready {
					break
				}
			}
			ready <- a
		}(a)
	}


	
	runCtx, runCancel := context.WithCancel(inputCtx)

	// Run each actor.
	runCh := make(chan error, len(g.actors))
	defer close(runCh)


	for _, a := range g.actors {
		go func(a actor) {
			<-ready
			runCh <- a.execute(runCtx)
		}(a)
	}

	// Wait for the first actor to stop.
	err := <-runCh

	// Notify Run() that is needs to stop.
	runCancel()

	var closeCtx context.Context
	{
		if g.closeTimeout == 0 {
			closeCtx = inputCtx
		} else {
			ctx, cancel := context.WithTimeout(inputCtx, g.closeTimeout)
			defer cancel()
			closeCtx = ctx
		}
	}

	// Notify Close() that it needs to stop.
	closeCh := make(chan struct{}, len(g.actors))
	defer close(closeCh)

	for _, a := range g.actors {
		a := a // NOTE(frank): May not need this anymore in go1.22.

		shutdown := func(a actor) {
			a.interrupt(closeCtx)
			closeCh <- struct{}{}
		}

		if g.syncShutdown {
			shutdown(a)
		} else {
			go shutdown(a)
		}
	}

	// Wait for all Close() to stop.
	for i := 0; i < cap(closeCh); i++ {
		<-closeCh
	}

	// Wait for all actors to stop.
	for i := 1; i < cap(runCh); i++ {
		<-runCh
	}

	// Return the original error.
	return err
}

type actor struct {
	execute   func(context.Context) error
	interrupt func(context.Context) error
	ready     func() bool
	id        ID
}
