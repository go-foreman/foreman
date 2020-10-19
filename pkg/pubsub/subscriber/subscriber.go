package subscriber

import (
	log "github.com/go-foreman/foreman/pkg/log"
	pubsubErr "github.com/go-foreman/foreman/pkg/pubsub/errors"
	"github.com/go-foreman/foreman/pkg/pubsub/transport/plugins/amqp"
	"os"
	"os/signal"
	"syscall"

	"context"
	"github.com/go-foreman/foreman/pkg/pubsub/transport"
	"github.com/go-foreman/foreman/pkg/pubsub/transport/pkg"
	"github.com/pkg/errors"
	"time"
)

const (
	maxTasksInProgress              = 100
	packageProcessingMaxTimeSeconds = 60
	gracefulShutdownTimeoutSeconds  = 120
)

type Subscriber interface {
	Run(ctx context.Context, queues ...transport.Queue) error
	Stop(ctx context.Context) error
}

type subscriber struct {
	transport        transport.Transport
	logger           log.Logger
	processor        Processor
	workerDispatcher *dispatcher
}

func NewSubscriber(transport transport.Transport, processor Processor, logger log.Logger) Subscriber {
	return &subscriber{transport: transport, logger: logger, processor: processor, workerDispatcher: newDispatcher(maxTasksInProgress)}
}

func (s *subscriber) Run(ctx context.Context, queues ...transport.Queue) error {
	s.logger.Logf(log.InfoLevel, "Started subscriber. Listening to queues: %v", queues)

	signalChan := make(chan os.Signal)
	defer close(signalChan)

	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	consumerCtx, cancelConsumerCtx := context.WithCancel(ctx)
	dispatcherCtx, cancelWorkers := context.WithCancel(ctx)
	shutdownCtx, _ := context.WithTimeout(context.Background(), time.Second*gracefulShutdownTimeoutSeconds)

	consumedPkgs, err := s.transport.Consume(consumerCtx, queues, amqp.WithQosPrefetchCount(maxTasksInProgress))

	if err != nil {
		return errors.WithStack(err)
	}

	s.workerDispatcher.start(dispatcherCtx)

	for {
		select {
		case incomingPkg, opened := <-consumedPkgs:
			if !opened {
				return nil
			}
			s.workerDispatcher.schedule(newTaskProcessPkg(ctx, incomingPkg, s))
		case <-ctx.Done():
			s.logger.Logf(log.InfoLevel, "Subscriber's context was canceled")
			if err := s.Stop(shutdownCtx); err != nil {
				s.logger.Logf(log.ErrorLevel, "Error stopping subscriber gracefully %s", err)
			}
			return nil
		case <-signalChan:
			s.logger.Logf(log.InfoLevel, "Received kill signal")
			cancelConsumerCtx()
			if err := s.Stop(shutdownCtx); err != nil {
				s.logger.Logf(log.ErrorLevel, "Error stopping subscriber gracefully %s", err)
			}
			cancelWorkers()
			return nil
		}
	}
}

func (s *subscriber) processPackage(ctx context.Context, inPkg pkg.IncomingPkg) {
	processorCtx, _ := context.WithTimeout(ctx, time.Second*packageProcessingMaxTimeSeconds)
	var toAck bool

	if err := s.processor.Process(processorCtx, inPkg); err != nil {
		s.logger.Logf(log.ErrorLevel, "error happened while processing pkg %s from %s. %s\n", inPkg.TraceId(), inPkg.Origin(), err)
		originalErr := errors.Cause(err)

		if statusErr, ok := originalErr.(pubsubErr.StatusErr); ok {
			if statusErr.Status == pubsubErr.NoRetry {
				toAck = true
			}
		}
	} else {
		toAck = true
	}

	if toAck {
		if err := inPkg.Ack(); err != nil {
			s.logger.Logf(log.ErrorLevel, "error acking package %s. %s", inPkg.TraceId(), err)
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
	ctx        context.Context
	pkg        pkg.IncomingPkg
	subscriber *subscriber
}

func newTaskProcessPkg(ctx context.Context, pkg pkg.IncomingPkg, subscriber *subscriber) *processPkg {
	return &processPkg{
		ctx:        ctx,
		pkg:        pkg,
		subscriber: subscriber,
	}
}

func (p *processPkg) do() {
	p.subscriber.processPackage(p.ctx, p.pkg)
}
