package messagebus

import (
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/dispatcher"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/endpoint"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message/execution"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/subscriber"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport/plugins/amqp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var ModuleFx = fx.Options(
	fx.Provide(func(logger *log.Logger) (transport.Transport, error) {
		return amqp.NewTransport(viper.GetString("amqp.connection"), logger), nil
	}),
	fx.Provide(endpoint.NewRouter),
	fx.Provide(dispatcher.NewDispatcher),
	fx.Provide(subscriber.NewHandler),
	fx.Provide(subscriber.NewSubscriber),
	fx.Provide(execution.NewMessageExecutionCtxFactory),
	fx.Provide(message.NewJsonDecoder),
)
