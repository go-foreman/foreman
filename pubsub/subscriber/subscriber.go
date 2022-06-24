package subscriber

import (
	"os"
	"os/signal"
	"syscall"

	"context"
	"time"

	"github.com/go-foreman/foreman/log"

	"github.com/go-foreman/foreman/pubsub/transport"
	"github.com/pkg/errors"
)

// Subscriber starts listening for queues and processes messages
type Subscriber interface {
	// Run listens queues for packages and processes them. Gracefully shuts down either on os.Signal or ctx.Done()
	Run(ctx context.Context, queues ...transport.Queue) error
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
	config      *Config
	consumeOpts []transport.ConsumeOpt
}

type Opt func(o *subscriberOpts)

func WithConfig(c *Config) Opt {
	return func(o *subscriberOpts) {
		o.config = c
	}
}

func WithConsumeOpts(opts ...transport.ConsumeOpt) Opt {
	return func(o *subscriberOpts) {
		o.consumeOpts = opts
	}
}

// NewSubscriber creates default subscriber implementation
func NewSubscriber(transport transport.Transport, processor Processor, logger log.Logger, opts ...Opt) Subscriber {
	sOpts := &subscriberOpts{}

	for _, o := range opts {
		o(sOpts)
	}

	if sOpts.config == nil {
		sOpts.config = &DefaultConfig
	}

	return &subscriber{
		transport:        transport,
		logger:           logger,
		processor:        processor,
		workerDispatcher: newDispatcher(sOpts.config.WorkersCount, logger),
		opts:             sOpts,
	}
}

type subscriber struct {
	transport        transport.Transport
	logger           log.Logger
	processor        Processor
	workerDispatcher *dispatcher
	opts             *subscriberOpts
}

func (s *subscriber) Run(ctx context.Context, queues ...transport.Queue) error {
	s.logger.Logf(log.InfoLevel, "Started subscriber. Listening to queues: %v", queues)

	ctx, cancelSignal := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancelSignal()

	config := s.opts.config

	consumerCtx, cancelConsumerCtx := context.WithCancel(ctx)

	consumedPkgs, err := s.transport.Consume(consumerCtx, queues, s.opts.consumeOpts...)

	if err != nil {
		cancelConsumerCtx()
		return errors.WithStack(err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), config.GracefulShutdownTimeout)
	defer shutdownCancel()

	defer s.gracefulShutdown(shutdownCtx)
	defer cancelConsumerCtx()

	s.workerDispatcher.start(consumerCtx)

	scheduleTicker := time.NewTicker(config.WorkerWaitingAssignmentTimeout)

	defer scheduleTicker.Stop()

	for {
		select {
		case <-consumerCtx.Done():
			s.logger.Log(log.InfoLevel, "Subscriber's context was canceled")
			return nil
		case worker, open := <-s.workerDispatcher.queue():
			if !open {
				s.logger.Log(log.InfoLevel, "worker's channel is closed")
				return nil
			}

			select {
			case <-scheduleTicker.C:
				s.logger.Logf(log.DebugLevel, "worker was waiting %s for a job to start. returning him to the pool", config.WorkerWaitingAssignmentTimeout.String())
				s.workerDispatcher.queue() <- worker
				break
			case incomingPkg, open := <-consumedPkgs:
				if !open {
					s.logger.Log(log.InfoLevel, "consumed package is closed")
					return nil
				}
				worker <- newTaskProcessPkg(ctx, incomingPkg, s, s.logger)
			}
		}
	}
}

func (s *subscriber) processPackage(ctx context.Context, inPkg transport.IncomingPkg) {
	processorCtx, processorCancel := context.WithTimeout(ctx, s.opts.config.PackageProcessingMaxTime)
	defer processorCancel()

	s.logger.Logf(log.DebugLevel, "started processing package id %s", inPkg.UID())

	if err := s.processor.Process(processorCtx, inPkg); err != nil {
		s.logger.Logf(log.ErrorLevel, "error happened while processing pkg %s from %s. %s", inPkg.UID(), inPkg.Origin(), err)

		return
	}

	if err := inPkg.Ack(); err != nil {
		s.logger.Logf(log.ErrorLevel, "error acking package %s. %s", inPkg.UID(), err)
		return
	}

	s.logger.Logf(log.DebugLevel, "acked package id %s", inPkg.UID())
}

func (s *subscriber) gracefulShutdown(ctx context.Context) {
	if s.workerDispatcher.busyWorkers() > 0 {
		s.logger.Logf(log.InfoLevel, "Graceful shutdown. Waiting subscriber for finishing %d tasks in progress", s.workerDispatcher.busyWorkers())
	}

	waitingTicker := time.NewTicker(time.Second)
	defer waitingTicker.Stop()

	for s.workerDispatcher.busyWorkers() > 0 {
		select {
		case <-ctx.Done():
			s.logger.Log(log.WarnLevel, "Stopped gracefulShutdown because of canceled parent ctx")
			return
		case <-waitingTicker.C:
			s.logger.Logf(log.InfoLevel, "Waiting for processor to finish all remaining tasks in a queue. Tasks in progress: %d", s.workerDispatcher.busyWorkers())
		}
	}

	s.logger.Log(log.InfoLevel, "All tasks are finished.")
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
