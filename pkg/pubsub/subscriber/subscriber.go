package subscriber

import (
	"context"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message"
	transport "github.com/kopaygorodsky/brigadier/pkg/pubsub/transport"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport/pkg"
	"github.com/pkg/errors"
	logger "github.com/sirupsen/logrus"
	"sync"
	"time"
)

const maxTasksInProgress = 1000

type Subscriber interface {
	Subscribe(ctx context.Context, queues []transport.Queue) error
	Stop(ctx context.Context) error
}

type subscriber struct {
	transport       transport.Transport
	logger          *logger.Logger
	processor       Processor
	tasksInProgress int
	sync.Mutex
}

func NewSubscriber(transport transport.Transport, processor Processor, logger *logger.Logger) Subscriber {
	return &subscriber{transport: transport, logger: logger, processor: processor}
}

func (s subscriber) Subscribe(ctx context.Context, queues []transport.Queue) error {
	s.logger.Infof("Started subscriber. Listening to queues: %v", queues)

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
				s.logger.Warningf("subscriber was waiting 10 seconds for a free spot to start processing package %s", incomingPkg.TraceId())
			}
		}

		go func(inPkg pkg.IncomingPkg) {
			s.Lock()
			s.tasksInProgress++
			s.Unlock()

			if err := s.processor.Process(ctx, inPkg); err != nil {
				s.logger.Errorf("Error happened while processing pkg %s from %s. %s\n", inPkg.TraceId(), inPkg.Origin(), err)

				switch errors.Cause(err).(type) {
				case message.DecoderErr:

					//todo configure DLX for such cases if you want to requeue message
					if err := inPkg.Nack(); err != nil {
						s.logger.Errorf("Error nacking package %s. %s", inPkg.TraceId(), err)
					}
				default:
					if err := inPkg.Ack(); err != nil {
						s.logger.Errorf("Error acking package %s. %s", inPkg.TraceId(), err)
					}
				}
			} else {
				if err := inPkg.Ack(); err != nil {
					s.logger.Errorf("Error acking package %s. %s", inPkg.TraceId(), err)
				}
			}

			s.Lock()
			s.tasksInProgress--
			s.Unlock()

		}(incomingPkg)
	}

	return nil

}

func (s subscriber) Stop(ctx context.Context) error {

	if s.tasksInProgress > 0 {
		s.logger.Infof("Graceful shutdown. Waiting subscriber for finishing %d tasks in progress", s.tasksInProgress)
	}

	for s.tasksInProgress > 0 {
		select {
		case <-ctx.Done():
			s.logger.Warn("Stopped subscriber because of canceled parent ctx")
			return nil
		case <-time.After(time.Second):
			{
				s.logger.Infof("Waiting for processor to finish all remaining tasks in a queue. Task in progress: %d", s.tasksInProgress)
			}
		}
	}

	s.logger.Infof("All tasks are finished. Disconnecting from transport.")

	return s.transport.Disconnect(ctx)
}
