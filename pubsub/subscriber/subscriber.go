package subscriber

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/go-foreman/foreman/log"
	"github.com/go-foreman/foreman/pubsub/transport/amqp"

	"context"
	"time"

	"github.com/go-foreman/foreman/pubsub/transport"
	"github.com/pkg/errors"
)

// Subscriber starts listening for queues and processes messages
type Subscriber interface {
	// Run listens queues for packages and processes them. Gracefully shuts down either on os.Signal or ctx.Done() or Stop()
	Run(ctx context.Context, queues ...transport.Queue) error
	// Stop gracefully stops subscriber and calls transport.Disconnect().
	Stop(ctx context.Context) error
}

// Config allows to configure subscriber workflow
type Config struct {
	// WorkersCount specifies a number workers that process packages
	WorkersCount uint
	// WorkerWaitingAssignmentTimeout amount of time that a worker will wait for assigning a package
	WorkerWaitingAssignmentTimeout time.Duration
	// PackageProcessingMaxTime amount of time for a package to be processed
	PackageProcessingMaxTime time.Duration
	// GracefulShutdownTimeout amount of time for graceful shutdown
	GracefulShutdownTimeout time.Duration
}

var DefaultConfig = Config{
	WorkersCount:                   10,
	WorkerWaitingAssignmentTimeout: time.Second * 3,
	PackageProcessingMaxTime:       time.Second * 60,
	GracefulShutdownTimeout:        time.Second * 61,
}

type subscriberOpts struct {
	config *Config
}

type Opt func(o *subscriberOpts)

func WithConfig(c *Config) Opt {
	return func(o *subscriberOpts) {
		o.config = c
	}
}

// NewSubscriber creates default subscriber implementation
func NewSubscriber(transport transport.Transport, processor Processor, logger log.Logger, opts ...Opt) Subscriber {
	sOpts := &subscriberOpts{}

	for _, o := range opts {
		o(sOpts)
	}

	var config *Config

	if sOpts.config != nil {
		config = sOpts.config
	} else {
		config = &DefaultConfig
	}

	return &subscriber{
		transport:        transport,
		logger:           logger,
		processor:        processor,
		workerDispatcher: newDispatcher(config.WorkersCount),
		config:           config,
	}
}

type subscriber struct {
	transport        transport.Transport
	logger           log.Logger
	processor        Processor
	workerDispatcher *dispatcher
	config           *Config
}

func (s *subscriber) Run(ctx context.Context, queues ...transport.Queue) error {
	s.logger.Logf(log.InfoLevel, "Started subscriber. Listening to queues: %v", queues)

	signalChan := make(chan os.Signal, 1)
	defer close(signalChan)

	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	consumerCtx, cancelConsumerCtx := context.WithCancel(ctx)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), s.config.GracefulShutdownTimeout)
	defer shutdownCancel()
	defer cancelConsumerCtx()

	consumedPkgs, err := s.transport.Consume(consumerCtx, queues, amqp.WithQosPrefetchCount(s.config.WorkersCount))

	if err != nil {
		return errors.WithStack(err)
	}

	s.workerDispatcher.start(consumerCtx)

	scheduleTicker := time.NewTicker(s.config.WorkerWaitingAssignmentTimeout)

	defer scheduleTicker.Stop()

	for {
		select {
		case worker, open := <-s.workerDispatcher.queue():
			if !open {
				s.logger.Logf(log.InfoLevel, "worker's channel is closed")
				return nil
			}
			select {
			case <-scheduleTicker.C:
				s.logger.Logf(log.DebugLevel, "worker was waiting %s for a job to start. returning him to the pool", s.config.WorkerWaitingAssignmentTimeout.String())
				s.workerDispatcher.queue() <- worker
				break
			case incomingPkg, open := <-consumedPkgs:
				if !open {
					return nil
				}
				worker <- newTaskProcessPkg(ctx, incomingPkg, s, s.logger)
			}
		case <-ctx.Done():
			s.logger.Logf(log.InfoLevel, "Subscriber's context was canceled")
			if err := s.Stop(shutdownCtx); err != nil {
				s.logger.Logf(log.ErrorLevel, "error stopping subscriber gracefully %s", err)
				return errors.Wrapf(err, "stopping subscriber gracefully")
			}
			return nil
		case <-signalChan:
			s.logger.Logf(log.InfoLevel, "Received kill signal")
			if err := s.Stop(shutdownCtx); err != nil {
				s.logger.Logf(log.ErrorLevel, "error stopping subscriber gracefully %s", err)
				return errors.Wrapf(err, "stopping subscriber gracefully")
			}
			return nil
		}
	}
}

func (s *subscriber) processPackage(ctx context.Context, inPkg transport.IncomingPkg) {
	processorCtx, processorCancel := context.WithTimeout(ctx, s.config.PackageProcessingMaxTime)
	defer processorCancel()

	s.logger.Logf(log.DebugLevel, "started processing package id %s", inPkg.UID())

	if err := s.processor.Process(processorCtx, inPkg); err != nil {
		s.logger.Logf(log.ErrorLevel, "error happened while processing pkg %s from %s. %s\n", inPkg.UID(), inPkg.Origin(), err)

		return
	}

	if err := inPkg.Ack(); err != nil {
		s.logger.Logf(log.ErrorLevel, "error acking package %s. %s", inPkg.UID(), err)
		return
	}

	s.logger.Logf(log.DebugLevel, "acked package id %s", inPkg.UID())
}

func (s *subscriber) Stop(ctx context.Context) error {
	if s.workerDispatcher.busyWorkers() > 0 {
		s.logger.Logf(log.InfoLevel, "Graceful shutdown. Waiting subscriber for finishing %d tasks in progress", s.workerDispatcher.busyWorkers())
	}

	waitingTicker := time.NewTicker(time.Second)
	defer waitingTicker.Stop()

	for s.workerDispatcher.busyWorkers() > 0 {
		select {
		case <-ctx.Done():
			s.logger.Logf(log.WarnLevel, "Stopped subscriber because of canceled parent ctx")
			return nil
		case <-waitingTicker.C:
			s.logger.Logf(log.InfoLevel, "Waiting for processor to finish all remaining tasks in a queue. Tasks in progress: %d", s.workerDispatcher.busyWorkers())
		}
	}

	s.logger.Logf(log.InfoLevel, "All tasks are finished. Disconnecting from transport.")

	return s.transport.Disconnect(ctx)
}

type processPkg struct {
	ctx        context.Context
	pkg        transport.IncomingPkg
	subscriber *subscriber
	logger     log.Logger
}

func newTaskProcessPkg(ctx context.Context, pkg transport.IncomingPkg, subscriber *subscriber, logger log.Logger) *processPkg {
	return &processPkg{
		ctx:        ctx,
		pkg:        pkg,
		subscriber: subscriber,
		logger:     logger,
	}
}

func (p *processPkg) do() {
	p.subscriber.processPackage(p.ctx, p.pkg)
}
