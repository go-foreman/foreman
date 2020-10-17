package subscriber

import (
	"context"
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

func (w *worker) workerQueue() workerQueue {
	return w.myTasks
}

func newWorker(ctx context.Context, dispatcherQueue dispatcherQueue) worker {
	return worker{
		ctx:             ctx,
		dispatcherQueue: dispatcherQueue,
		myTasks:         make(workerQueue),
	}
}

func (w *worker) start() {
	go func() {
		defer close(w.myTasks)
		for {
			//tell dispatcher that I'm ready to work
			w.dispatcherQueue <- w.myTasks
			select {
			case <-w.ctx.Done():
				return
			case task, open := <-w.myTasks:
				if !open {
					return
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
	}
}

type dispatcher struct {
	ctx           context.Context
	workersCount  uint
	workersQueues dispatcherQueue
}

func (d *dispatcher) busyWorkers() int {
	return int(d.workersCount) - len(d.workersQueues)
}

func (d *dispatcher) start(ctx context.Context) {
	d.ctx = ctx

	var i uint
	for i < d.workersCount {
		workerCtx, _ := context.WithCancel(ctx)
		worker := newWorker(workerCtx, d.workersQueues)
		worker.start()
		i++
	}
}

func (d dispatcher) schedule(task task) {
	for {
		select {
		case <-d.ctx.Done():
			return
		case worker, opened := <-d.workersQueues:
			if !opened {
				return
			}
			worker <- task
			return
		}
	}
}
