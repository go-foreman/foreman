package subscriber

import (
	"context"
	"sync"
)

type task interface {
	do()
}

type dispatcherQueue chan chan task
type workerQueue chan task

type worker struct {
	ctx             context.Context
	dispatcherQueue dispatcherQueue
	myTasks         workerQueue
}

func newWorker(ctx context.Context, dispatcherQueue dispatcherQueue) worker {
	return worker{
		ctx:             ctx,
		dispatcherQueue: dispatcherQueue,
		myTasks:         make(workerQueue),
	}
}

func (w *worker) start(wGroup *sync.WaitGroup) {
	go func() {
		defer wGroup.Done()
		defer close(w.myTasks)
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

func newDispatcher(workersCount uint) *dispatcher {
	return &dispatcher{
		workersCount:  workersCount,
		workersQueues: make(dispatcherQueue, workersCount),
		mutex:         &sync.RWMutex{},
	}
}

type dispatcher struct {
	mutex *sync.RWMutex

	stopped       bool
	workersCount  uint
	workersQueues dispatcherQueue
}

// busyWorkers return number of workers that are busy with processing a task and weren't returned to the dispatcher
func (d *dispatcher) busyWorkers() int {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if d.stopped {
		return 0
	}

	return int(d.workersCount) - len(d.workersQueues)
}

// start schedules defined number of workers
func (d *dispatcher) start(ctx context.Context) {
	wGroup := &sync.WaitGroup{}
	var i uint

	workersCtx, stopWorkers := context.WithCancel(ctx)

	for i = 0; i < d.workersCount; i++ {
		worker := newWorker(workersCtx, d.workersQueues)
		wGroup.Add(1)
		worker.start(wGroup)
	}

	go func() {
		<-ctx.Done()

		// Dry out all workers from the pool. The loop over workersCount is used very much intentionally.
		// After each worker finishes it's job it places itself into the pool again and that's where we have a chance to catch it and remove from the pool.
		// Eventually all d.workersCount must be removed from the pool before we can close the pool and cancel workers ctx
		for i := 0; i < int(d.workersCount); i++ {
			<-d.workersQueues
		}

		// close the pool
		close(d.workersQueues)

		stopWorkers()

		// wait for all workers to stop
		wGroup.Wait()

		d.mutex.Lock()
		d.stopped = true
		d.mutex.Unlock()
	}()
}

// queue returns worker's chan that is ready to accept a job to do
// @todo making this method's access by value (without *) causes race detector to detect a race at writing d.stopped = true in .start() method. waaaaaat? Go 1.18.3
func (d *dispatcher) queue() dispatcherQueue {
	return d.workersQueues
}
