package worker

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type Task func()

type WorkerPool struct {
	tasks         chan Task
	wg            sync.WaitGroup
	activeWorkers int
	maxWorkers    int
	logger        zerolog.Logger
	mu            sync.RWMutex
	shutdown      chan struct{}
}

func NewWorkerPool(maxWorkers int, logger zerolog.Logger) *WorkerPool {
	return &WorkerPool{
		tasks:      make(chan Task, maxWorkers*10),
		maxWorkers: maxWorkers,
		logger:     logger,
		shutdown:   make(chan struct{}),
	}
}

func (wp *WorkerPool) Start(ctx context.Context) error {
	wp.logger.Info().Int("max_workers", wp.maxWorkers).Msg("Starting worker pool")

	for i := 0; i < wp.maxWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}

	wp.logger.Info().Int("workers_started", wp.maxWorkers).Msg("Worker pool started")
	return nil
}

func (wp *WorkerPool) Stop() error {
	wp.logger.Info().Msg("Stopping worker pool")

	close(wp.tasks)

	wp.wg.Wait()

	close(wp.shutdown)

	wp.logger.Info().Msg("Worker pool stopped")
	return nil
}

func (wp *WorkerPool) Submit(task Task) {
	select {
	case wp.tasks <- task:
	default:
		wp.logger.Warn().Msg("Worker pool task queue is full")
		select {
		case wp.tasks <- task:
		case <-time.After(1 * time.Second):
			wp.logger.Error().Msg("Failed to submit task to worker pool (timeout)")
		}
	}
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	wp.mu.Lock()
	wp.activeWorkers++
	wp.mu.Unlock()

	wp.logger.Debug().Int("worker_id", id).Msg("Worker started")

	for task := range wp.tasks {
		wp.logger.Debug().Int("worker_id", id).Msg("Worker processing task")

		wp.mu.Lock()
		wp.activeWorkers--
		wp.mu.Unlock()

		func() {
			defer func() {
				if r := recover(); r != nil {
					wp.logger.Error().
						Int("worker_id", id).
						Interface("panic", r).
						Msg("Worker recovered from panic")
				}

				wp.mu.Lock()
				wp.activeWorkers++
				wp.mu.Unlock()
			}()

			task()
		}()
	}

	wp.mu.Lock()
	wp.activeWorkers--
	wp.mu.Unlock()

	wp.logger.Debug().Int("worker_id", id).Msg("Worker stopped")
}

func (wp *WorkerPool) GetActiveWorkers() int {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.activeWorkers
}

func (wp *WorkerPool) GetQueueLength() int {
	return len(wp.tasks)
}

func (wp *WorkerPool) GetStats() map[string]interface{} {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	return map[string]interface{}{
		"active_workers": wp.activeWorkers,
		"max_workers":    wp.maxWorkers,
		"queue_length":   len(wp.tasks),
		"queue_capacity": cap(wp.tasks),
	}
}
