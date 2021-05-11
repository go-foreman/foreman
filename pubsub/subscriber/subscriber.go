package subscriber

import (
	"github.com/go-foreman/foreman/log"
	pubsubErr "github.com/go-foreman/foreman/pubsub/errors"
	"github.com/go-foreman/foreman/pubsub/transport/plugins/amqp"
	"os"
	"os/signal"
	"syscall"

	"context"
	"github.com/go-foreman/foreman/pubsub/transport"
	"github.com/go-foreman/foreman/pubsub/transport/pkg"
	"github.com/pkg/errors"
	"time"
)

const (
	maxTasksInProgress                     = 100
	packageProcessingMaxTime time.Duration = time.Second * 60
	gracefulShutdownTimeout  time.Duration = time.Second * 120
	scheduleTimeout          time.Duration = time.Second
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
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer shutdownCancel()
	defer cancelConsumerCtx()

	consumedPkgs, err := s.transport.Consume(consumerCtx, queues, amqp.WithQosPrefetchCount(maxTasksInProgress))

	if err != nil {
		return errors.WithStack(err)
	}

	s.workerDispatcher.start(consumerCtx)

	testCtx, _ := context.WithTimeout(context.Background(), time.Second * 10)

	scheduleTicker := time.NewTicker(scheduleTimeout)

	defer scheduleTicker.Stop()

	for {
		select {
		case worker, open := <- s.workerDispatcher.readyWorker():
			if !open {
				s.logger.Logf(log.InfoLevel, "worker's channel is closed")
				return nil
			}
			select {
			case <- scheduleTicker.C:
				s.logger.Logf(log.InfoLevel, "worker was waiting %s for a job to start. returning to main loop", scheduleTimeout.String())
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
			}
			return nil
		case <-signalChan:
			s.logger.Logf(log.InfoLevel, "Received kill signal")
			if err := s.Stop(shutdownCtx); err != nil {
				s.logger.Logf(log.ErrorLevel, "error stopping subscriber gracefully %s", err)
			}
			return nil
		case <- testCtx.Done():
			s.logger.Logf(log.InfoLevel, "Test kill")
			if err := s.Stop(shutdownCtx); err != nil {
				s.logger.Logf(log.ErrorLevel, "error stopping subscriber gracefully %s", err)
			}
			return nil
		}
	}
}

func (s *subscriber) processPackage(ctx context.Context, inPkg pkg.IncomingPkg) {
	processorCtx, processorCancel := context.WithTimeout(ctx, packageProcessingMaxTime)
	defer processorCancel()

	var toAck bool

	if err := s.processor.Process(processorCtx, inPkg); err != nil {
		s.logger.Logf(log.ErrorLevel, "error happened while processing pkg %s from %s. %s\n", inPkg.UID(), inPkg.Origin(), err)
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
			s.logger.Logf(log.ErrorLevel, "error acking package %s. %s", inPkg.UID(), err)
		}
	}
}

func (s *subscriber) Stop(ctx context.Context) error {
	if s.workerDispatcher.busyWorkers() > 0 {
		s.logger.Logf(log.InfoLevel, "Graceful shutdown. Waiting subscriber for finishing %d tasks in progress", s.workerDispatcher.busyWorkers())
	}

	waitingTicker := time.NewTicker(time.Second)
	defer waitingTicker.Stop()

	for s.workerDispatcher.busyWorkers() > 0 {
		select {
		case <- ctx.Done():
			s.logger.Logf(log.WarnLevel, "Stopped subscriber because of canceled parent ctx")
			return nil
		case <- waitingTicker.C:
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
	logger log.Logger
}

func newTaskProcessPkg(ctx context.Context, pkg pkg.IncomingPkg, subscriber *subscriber, logger log.Logger) *processPkg {
	return &processPkg{
		ctx:        ctx,
		pkg:        pkg,
		subscriber: subscriber,
		logger: logger,
	}
}

func (p *processPkg) do() {
	p.subscriber.processPackage(p.ctx, p.pkg)
}

//func (p *processPkg) do() {
//	p.logger.Logf(log.InfoLevel, "started %s", p.pkg.UID())
//	ctx, cancel := context.WithCancel(p.ctx)
//	defer cancel()
//
//	go func() {
//		defer cancel()
//		p.subscriber.processPackage(p.ctx, p.pkg)
//	}()
//
//	processingTimer := time.NewTicker(time.Second*2)
//	defer processingTimer.Stop()
//
//	for {
//		select {
//		case <- ctx.Done():
//			p.logger.Logf(log.InfoLevel, "finished %s", p.pkg.UID())
//			return
//		case <- processingTimer.C:
//			p.logger.Logf(log.InfoLevel, "seems job is hung %s", p.pkg.UID())
//		}
//	}
//}
