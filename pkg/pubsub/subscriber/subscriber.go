package subscriber

import (
	log "github.com/kopaygorodsky/brigadier/pkg/log"

	"context"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport/pkg"
	"github.com/pkg/errors"
	"sync"
	"time"
)

const maxTasksInProgress = 1000

type Subscriber interface {
	Subscribe(ctx context.Context, queues... transport.Queue) error
	Stop(ctx context.Context) error
}

type subscriber struct {
	transport       transport.Transport
	logger          log.Logger
	processor       Processor
	tasksInProgress int
	sync.Mutex
}

func NewSubscriber(transport transport.Transport, processor Processor, logger log.Logger) Subscriber {
	return &subscriber{transport: transport, logger: logger, processor: processor}
}

func (s *subscriber) Subscribe(ctx context.Context, queues... transport.Queue) error {
	s.logger.Logf(log.InfoLevel, "Started subscriber. Listening to queues: %v", queues)

	consumedPkgs, err := s.transport.Consume(ctx, queues)

	if err != nil {
		return errors.WithStack(err)
	}

	for incomingPkg := range consumedPkgs {

		//waiting until there is free spot to start processing package
		for s.tasksInProgress >= maxTasksInProgress {
			select {
			case <-time.After(time.Second):
				{

				}
			case <-time.After(time.Second * 10):
				s.logger.Logf(log.WarnLevel, "subscriber was waiting 10 seconds for a free spot to start processing package %s", incomingPkg.TraceId())
			}
		}

		go func(inPkg pkg.IncomingPkg) {
			s.Lock()
			s.tasksInProgress++
			s.Unlock()

			if err := s.processor.Process(ctx, inPkg); err != nil {
				s.logger.Logf(log.ErrorLevel, "Error happened while processing pkg %s from %s. %s\n", inPkg.TraceId(), inPkg.Origin(), err)

				//switch errors.Cause(err).(type) {
				//case message.DecoderErr:
				//
				//	//todo configure DLX for such cases if you want to requeue message
				//	if err := inPkg.Nack(); err != nil {
				//		s.logger.Logf(log.ErrorLevel, "Error nacking package %s. %s", inPkg.TraceId(), err)
				//	}
				//default:
				//	s.logger.Logf(log.ErrorLevel, "Error acking package %s. %s", inPkg.TraceId(), err)
				//}
			} else {
				if err := inPkg.Ack(); err != nil {
					s.logger.Logf(log.ErrorLevel, "Error acking package %s. %s", inPkg.TraceId(), err)
				}
			}

			s.Lock()
			s.tasksInProgress--
			s.Unlock()

		}(incomingPkg)
	}

	return nil

}

func (s *subscriber) Stop(ctx context.Context) error {

	if s.tasksInProgress > 0 {
		s.logger.Logf(log.InfoLevel, "Graceful shutdown. Waiting subscriber for finishing %d tasks in progress", s.tasksInProgress)
	}

	for s.tasksInProgress > 0 {
		select {
		case <-ctx.Done():
			s.logger.Logf(log.WarnLevel, "Stopped subscriber because of canceled parent ctx")
			return nil
		case <-time.After(time.Second):
			{
				s.logger.Logf(log.InfoLevel, "Waiting for processor to finish all remaining tasks in a queue. Task in progress: %d", s.tasksInProgress)
			}
		}
	}

	s.logger.Logf(log.InfoLevel, "All tasks are finished. Disconnecting from transport.")

	return s.transport.Disconnect(ctx)
}
