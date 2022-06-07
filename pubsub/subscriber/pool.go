package subscriber

import (
	"context"
	"sync"

	"github.com/go-foreman/foreman/log"
)

type task interface {
	do()
}

type dispatcherQueue chan chan task
type workerQueue chan task

type worker struct {
	id              uint
	ctx             context.Context
	dispatcherQueue dispatcherQueue
	myTasks         workerQueue
	logger          log.Logger
}

func newWorker(id uint, ctx context.Context, dispatcherQueue dispatcherQueue, logger log.Logger) worker {
	return worker{
		id:              id,
		ctx:             ctx,
		dispatcherQueue: dispatcherQueue,
		myTasks:         make(workerQueue),
		logger:          logger,
	}
}

func (w *worker) start(wGroup *sync.WaitGroup, startCall func()) {
	go func() {
		defer wGroup.Done()
		defer close(w.myTasks)

		startCall()

		w.logger.Logf(log.DebugLevel, "worker %d started", w.id)

		for {
			w.dispatcherQueue <- w.myTasks

			select {
			case <-w.ctx.Done():
				return
			case task, open := <-w.myTasks:
				if !open {
					panic("someone explicitly closed the channel of this worker")
				}
				task.do()
			}
		}
	}()
}

func newDispatcher(workersCount uint, logger log.Logger) *dispatcher {
	return &dispatcher{
		workersCount:  workersCount,
		workersQueues: make(dispatcherQueue, workersCount),
		mutex:         &sync.RWMutex{},
		logger:        logger,
	}
}

type dispatcher struct {
	mutex *sync.RWMutex

	startedWorkers uint
	stoppedWorkers uint
	workersCount   uint
	workersQueues  dispatcherQueue
	logger         log.Logger
}

// busyWorkers return number of workers that are busy with processing a task and weren't returned to the dispatcher
func (d *dispatcher) busyWorkers() uint {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if d.stoppedWorkers == d.workersCount {
		return 0
	}

	// means not all workers started
	if d.startedWorkers < d.workersCount {
		return d.startedWorkers - uint(len(d.workersQueues))
	}

	return d.workersCount - uint(len(d.workersQueues)) - d.stoppedWorkers
}

// start schedules defined number of workers
func (d *dispatcher) start(ctx context.Context) {
	wGroup := &sync.WaitGroup{}
	var i uint

	workersCtx, stopWorkers := context.WithCancel(context.Background())

	for i = 0; i < d.workersCount; i++ {
		worker := newWorker(i, workersCtx, d.workersQueues, d.logger)
		wGroup.Add(1)
		worker.start(wGroup, func() {
			d.mutex.Lock()
			d.startedWorkers++
			d.mutex.Unlock()
		})
	}

	go func() {
		<-ctx.Done()

		// Dry out all workers from the pool. The loop over workersCount is used very much intentionally.
		// After each worker finishes it's job it places itself into the pool again and that's where we have a chance to catch it and remove from the pool.
		// Eventually all d.workersCount must be removed from the pool before we can close the pool and cancel workers ctx
		for i := 0; i < int(d.workersCount); i++ {
			<-d.workersQueues
			d.mutex.Lock()
			d.stoppedWorkers++
			d.mutex.Unlock()
		}

		// close the pool
		close(d.workersQueues)

		stopWorkers()

		// wait for all workers to stop
		wGroup.Wait()
	}()
}

// queue returns worker's chan that is ready to accept a job to do
// @todo making this method's access by value (without *) causes race detector to detect a race at writing d.stopped = true in .start() method. waaaaaat? Go 1.18.3
func (d *dispatcher) queue() dispatcherQueue {
	return d.workersQueues
}
