// Package worker drains the render queue.
package worker

import (
	"context"
	"errors"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"earthmanlabs/resumaker/internal/app"
	"earthmanlabs/resumaker/internal/store"
	"earthmanlabs/resumaker/render"
)

type Options struct {
	// Concurrency is the number of jobs processed at once. Defaults to 1.
	Concurrency int
	// PollInterval is how long an idle worker waits before checking for work.
	PollInterval time.Duration
	// StaleAfter is how old a running job must be before startup requeues it.
	StaleAfter time.Duration
}

const userErrorMessage = "Rendering failed. Try again."

// Run processes jobs until ctx is cancelled. It returns once every worker has stopped.
func Run(ctx context.Context, svc *app.Service, renderer *render.Renderer, opts Options) error {
	if opts.Concurrency < 1 {
		opts.Concurrency = 1
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 2 * time.Second
	}
	if opts.StaleAfter <= 0 {
		opts.StaleAfter = 10 * time.Minute
	}

	if n, err := svc.RequeueStaleRenderJobs(ctx, opts.StaleAfter); err != nil {
		log.Printf("worker: requeue stale jobs: %v", err)
	} else if n > 0 {
		log.Printf("worker: requeued %d stale job(s)", n)
	}

	var wg sync.WaitGroup
	for i := 0; i < opts.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if !runOne(ctx, svc, renderer, opts.PollInterval) {
					return
				}
			}
		}()
	}
	log.Printf("worker: started %d worker(s)", opts.Concurrency)

	<-ctx.Done()
	wg.Wait()
	return nil
}

// runOne processes at most one job and reports whether the worker should continue.
func runOne(ctx context.Context, svc *app.Service, renderer *render.Renderer, poll time.Duration) bool {
	job, err := svc.ClaimRenderJob(ctx)
	if errors.Is(err, store.ErrNotFound) {
		return wait(ctx, poll)
	}
	if err != nil {
		log.Printf("worker: claim job: %v", err)
		return wait(ctx, poll)
	}

	if err := process(ctx, svc, renderer, job); err != nil {
		log.Printf("worker: job %s failed: %v", job.ID, err)
		// Record the failure even if the job's context was cancelled.
		failCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := svc.FailRenderJob(failCtx, job.ID, userErrorMessage); err != nil {
			log.Printf("worker: mark job %s failed: %v", job.ID, err)
		}
	}
	return true
}

func process(ctx context.Context, svc *app.Service, renderer *render.Renderer, job store.ClaimedRenderJob) (err error) {
	// A panic in rendering must not take the worker down with a job stuck in "running".
	defer func() {
		if r := recover(); r != nil {
			log.Printf("worker: job %s panicked: %v\n%s", job.ID, r, debug.Stack())
			err = errors.New("render panicked")
		}
	}()
	return svc.ProcessRenderJob(ctx, job, renderer)
}

func wait(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
