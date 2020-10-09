package subscriber

import (
	log "github.com/kopaygorodsky/brigadier/pkg/log"
	"os"
	"os/signal"
	"syscall"

	"context"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport/pkg"
	"github.com/pkg/errors"
	"sync"
	"time"
)

const maxTasksInProgress = 100

type Subscriber interface {
	Subscribe(ctx context.Context, queues ...transport.Queue) error
	Stop(ctx context.Context) error
}

type subscriber struct {
	transport       transport.Transport
	logger          log.Logger
	processor       Processor
	workerDispatcher *Dispatcher
}

func NewSubscriber(transport transport.Transport, processor Processor, logger log.Logger) Subscriber {
	return &subscriber{transport: transport, logger: logger, processor: processor, workerDispatcher: newDispatcher(maxTasksInProgress)}
}

func (s *subscriber) Subscribe(ctx context.Context, queues ...transport.Queue) error {
	s.logger.Logf(log.InfoLevel, "Started subscriber. Listening to queues: %v", queues)

	consumedPkgs, err := s.transport.Consume(ctx, queues)

	if err != nil {
		return errors.WithStack(err)
	}

	var waitFor sync.WaitGroup
	waitFor.Add(1)

	dispatcherCtx, stopWorkers := context.WithCancel(ctx)
	go func() {
		defer waitFor.Done()

		signalChan := make(chan os.Signal)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
		<-signalChan
		stopWorkers()
		shutdownCtx, _ := context.WithTimeout(context.Background(), time.Second * 300)
		if err := s.Stop(shutdownCtx); err != nil {
			s.logger.Logf(log.ErrorLevel, "Error stopping subscriber gracefully %s", err)
		}
	}()

	s.workerDispatcher.start(dispatcherCtx)

	for incomingPkg := range consumedPkgs {
		s.workerDispatcher.schedule(newTaskProcessPkg(ctx, incomingPkg, s))
	}

	waitFor.Wait()

	return nil

}

func (s *subscriber) processPackage(ctx context.Context, inPkg pkg.IncomingPkg) {
	if err := s.processor.Process(ctx, inPkg); err != nil {
		s.logger.Logf(log.ErrorLevel, "Error happened while processing pkg %s from %s. %s\n", inPkg.TraceId(), inPkg.Origin(), err)
	} else {
		if err := inPkg.Ack(); err != nil {
			s.logger.Logf(log.ErrorLevel, "Error acking package %s. %s", inPkg.TraceId(), err)
		}
	}
}

func (s *subscriber) Stop(ctx context.Context) error {
	if s.workerDispatcher.busyWorkers() > 0 {
		s.logger.Logf(log.InfoLevel, "Graceful shutdown. Waiting subscriber for finishing %d tasks in progress", s.workerDispatcher.busyWorkers())
	}

	for s.workerDispatcher.busyWorkers() > 0 {
		select {
		case <-ctx.Done():
			s.logger.Logf(log.WarnLevel, "Stopped subscriber because of canceled parent ctx")
			return nil
		case <-time.After(time.Second):
			s.logger.Logf(log.InfoLevel, "Waiting for processor to finish all remaining tasks in a queue. Tasks in progress: %d", s.workerDispatcher.busyWorkers())
		}
	}

	s.logger.Logf(log.InfoLevel, "All tasks are finished. Disconnecting from transport.")

	return s.transport.Disconnect(ctx)
}

type processPkg struct {
	ctx context.Context
	pkg pkg.IncomingPkg
	subscriber *subscriber
}

func newTaskProcessPkg(ctx context.Context, pkg pkg.IncomingPkg, subscriber *subscriber) *processPkg {
	return &processPkg{
		ctx:        ctx,
		pkg:        pkg,
		subscriber: subscriber,
	}
}

func(p *processPkg) do() {
	p.subscriber.processPackage(p.ctx, p.pkg)
}
