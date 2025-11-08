package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"queuectl/internal/job"
)

// Pool manages a pool of workers
type Pool struct {
	workerCount int
	workers     []*Worker
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
}

// Worker represents a single worker goroutine
type Worker struct {
	id          int
	pool        *Pool
	running     bool
	currentJob  *job.Job
	mu          sync.Mutex
}

var globalPool *Pool

// StartPool starts a worker pool with the specified number of workers
func StartPool(count int) error {
	if globalPool != nil && globalPool.IsRunning() {
		return fmt.Errorf("worker pool is already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	pool := &Pool{
		workerCount: count,
		workers:     make([]*Worker, count),
		ctx:         ctx,
		cancel:      cancel,
	}

	for i := 0; i < count; i++ {
		worker := &Worker{
			id:   i + 1,
			pool: pool,
		}
		pool.workers[i] = worker
		pool.wg.Add(1)
		go worker.run()
	}

	globalPool = pool
	return nil
}

// StopPool stops all workers gracefully
func StopPool() error {
	if globalPool == nil || !globalPool.IsRunning() {
		return fmt.Errorf("worker pool is not running")
	}

	globalPool.cancel()
	globalPool.wg.Wait()
	globalPool = nil
	return nil
}

// GetPool returns the global worker pool
func GetPool() *Pool {
	return globalPool
}

// IsRunning returns whether the pool is currently running
func (p *Pool) IsRunning() bool {
	if p == nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ctx.Err() == nil
}

// GetWorkerCount returns the number of active workers
func (p *Pool) GetWorkerCount() int {
	if p == nil {
		return 0
	}
	return p.workerCount
}

// run is the main worker loop
func (w *Worker) run() {
	defer w.pool.wg.Done()

	w.pool.mu.Lock()
	w.running = true
	w.pool.mu.Unlock()

	for {
		// Check for shutdown signal
		select {
		case <-w.pool.ctx.Done():
			// Graceful shutdown - finish current job if any
			w.mu.Lock()
			currentJob := w.currentJob
			w.mu.Unlock()
			
			if currentJob != nil {
				// Job is currently executing (ExecuteJob is blocking)
				// It will finish before we can exit, which is correct behavior
				fmt.Printf("Worker %d: Finishing current job %s before shutdown...\n", w.id, currentJob.ID)
			}
			
			w.pool.mu.Lock()
			w.running = false
			w.pool.mu.Unlock()
			return
		default:
		}

		// Try to get next job
		j, err := GetNextJob()
		if err != nil {
			fmt.Printf("Worker %d: Error getting next job: %v\n", w.id, err)
			time.Sleep(1 * time.Second)
			continue
		}

		if j == nil {
			// No jobs available, wait a bit
			time.Sleep(1 * time.Second)
			continue
		}

		// Track current job
		w.mu.Lock()
		w.currentJob = j
		w.mu.Unlock()

		// Execute the job (blocking call - if shutdown is requested during execution,
		// this will complete first, then we'll check ctx.Done() on next iteration)
		if err := ExecuteJob(j); err != nil {
			fmt.Printf("Worker %d: Error executing job %s: %v\n", w.id, j.ID, err)
		} else {
			fmt.Printf("Worker %d: Completed job %s\n", w.id, j.ID)
		}

		// Clear current job
		w.mu.Lock()
		w.currentJob = nil
		w.mu.Unlock()

		// Check for shutdown after job execution (don't pick up new job if shutting down)
		select {
		case <-w.pool.ctx.Done():
			w.pool.mu.Lock()
			w.running = false
			w.pool.mu.Unlock()
			return
		default:
		}
	}
}

