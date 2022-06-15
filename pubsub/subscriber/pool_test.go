package subscriber

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorkerPool(t *testing.T) {
	rand.Seed(time.Now().Unix())

	t.Run("100 workers", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		workersCount := 10
		jobsChan := simulateConsume(100)
		workersDispatcher := newDispatcher(uint(workersCount))

		workersDispatcher.start(ctx)

		time.Sleep(time.Second)

		assert.Equal(t, 0, workersDispatcher.busyWorkers())

		counter := 0
		for job := range jobsChan {
			// verify that all workers are getting busy in order 1 2 3 4...
			if counter < workersCount {
				assert.Equal(t, counter, workersDispatcher.busyWorkers())
			}

			worker := <-workersDispatcher.queue()
			worker <- job
			//t.Logf("Job %d assigned, busy workers %d", job.id, workersDispatcher.busyWorkers())
			counter++
		}

		time.Sleep(time.Second)

		assert.Equal(t, 0, workersDispatcher.busyWorkers())
	})

	t.Run("cancel ctx", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		workersCount := 10
		workersDispatcher := newDispatcher(uint(workersCount))
		workersDispatcher.start(ctx)

		time.Sleep(time.Millisecond * 500)

		//cancel dispatcher's context, all workers finish their jobs and stop
		cancel()

		for i := 1; i < 1000; i++ {
			worker, opened := <-workersDispatcher.queue()
			if !opened {
				t.Logf("%d jobs processed after ctx was cancelled", i)
				break
			}
			worker <- &job{id: i}
		}

		time.Sleep(time.Millisecond * 200)

		_, opened := <-workersDispatcher.queue()
		assert.False(t, opened)

		assert.Equal(t, 0, workersDispatcher.busyWorkers())
	})

	//t.Run("catch panic when someone closed a worker's channel explicitly", func(t *testing.T) {
	//	ctx, cancel := context.WithCancel(context.Background())
	//	defer cancel()
	//
	//	workersCount := 10
	//
	//	assert.Panics(t, func() {
	//		workersDispatcher := newDispatcher(uint(workersCount))
	//		workersDispatcher.start(ctx)
	//
	//		time.Sleep(time.Millisecond * 500)
	//
	//		w := <-workersDispatcher.queue()
	//
	//		time.Sleep(time.Millisecond * 100)
	//		close(w)
	//	})
	//})

}

type job struct {
	id int
}

func (j job) do() {
	if rand.Intn(10)%4 == 0 {
		time.Sleep(time.Millisecond * 800)
		return
	}

	// between 200 and 400ms
	time.Sleep(time.Millisecond * time.Duration(rand.Intn(200)+200))
}

func simulateConsume(jobsCount int) chan *job {
	respChan := make(chan *job)

	go func() {
		defer close(respChan)
		for i := 0; i < jobsCount; i++ {
			respChan <- &job{id: i}
		}
	}()

	return respChan
}
