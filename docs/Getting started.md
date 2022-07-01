# Getting started

In order to start working with message bus one needs to prepare transport layer, in this example amqp will be used.

Need to connect to amqp transport and, depending on requirements, create a queue, topic and bind them together. Subscriber is able to listen to multiple queues at the same time.

```go
// user can implement own logger implementation for the log.Logger interface
defaultLogger := log.DefaultLogger(os.Stdout)

conn, _ := foremanAmqp.Dial("amqp://admin:admin123@127.0.0.1:5673", true, defaultLogger)

// creating new AMQP transport
amqpTransport := foremanAmqp.NewTransport(conn, defaultLogger)

defer func() {
    if err := conn.Close(); err != nil {
        defaultLogger.Log(log.ErrorLevel, err.Error())
    }
}()

// creating queue definition with options
queue := foremanAmqp.Queue(queueName, false, false, false, false)
// creating topic(exchange) definition with options
topic := foremanAmqp.Topic(topicName, false, false, false, false)
// binding the queue to the topic
binds := foremanAmqp.QueueBind(topic.Name(), fmt.Sprintf("%s.#", topic.Name()), false)

ctx := context.Background()

// create topic if such does not exist
if err := amqpTransport.CreateTopic(ctx, topic); err != nil {
    defaultLogger.Logf(log.ErrorLevel, "Error creating topic %s. %s", topic.Name(), err)
    panic(err)
}

// create queue if one does not exist and bind it
if err := amqpTransport.CreateQueue(ctx, queue, binds); err != nil {
    defaultLogger.Logf(log.ErrorLevel, "Error creating queue %s. %s", queue.Name(), err)
    panic(err)
}
```

Create an instance of shema and marshaller

```go
// it's possible to use global scheme instance on package level or create new one using constructor
schemeRegistry := scheme.KnownTypesRegistryInstance
// marshaller is responsible for encoding/decoding messages
marshaller := message.NewJsonMarshaller(schemeRegistry)
```

Create MessageBus instance with its dependencies

```go
// creating an instance of the message bus with all its dependencies.
bus, err := foreman.NewMessageBus(
    defaultLogger,
    marshaller,
    schemeRegistry,
    foreman.DefaultSubscriber(amqpTransport), // create default subscriber 
)
```

In order to reply with messages in executors(handlers) an endpoint is needed, messages must be registered onto one.

```go
// an endpoint is used by execution context when a user wants to call execCtx.Send().
amqpEndpoint := endpoint.NewAmqpEndpoint(
    fmt.Sprintf("%s_endpoint", queue.Name()),
    amqpTransport,
    transport.DeliveryDestination{
        DestinationTopic: topic.Name(),
        RoutingKey:       fmt.Sprintf("%s.eventAndCommands", topic.Name()),
    },
    marshaller, //endpoint encodes a message before sending it
)
```

Final step: registering and subscribing message types in schema, dispatcher and router

```go
//here all registrations are happening...

//all types that go through message bus must be registered in schema, it's suggested to do it in init() function
bus.SchemeRegistry().AddKnownTypes(scheme.Group("some"), &SomeCommand{}, &SomeEvent{})

//subscribe handler for its command and event
h := &Handler{}
bus.Dispatcher().SubscribeForCmd(&SomeCommand{}, h.handleSomeCommand)
bus.Dispatcher().SubscribeForEvent(&SomeEvent{}, h.handleSomeEvent)

//subscribe both messages for amqp endpoint, so execution context will know where to send replies with these types
bus.Router().RegisterEndpoint(amqpEndpoint, &SomeEvent{}, &SomeCommand{})
```

And start the subscriber `bus.Subscriber().Run(ctx, queue)`

Handlers & messages

```go
type SomeCommand struct {
   message.ObjectMeta //all types must have embedded ObjectMeta
   MyID string `json:"my_id"`
}

type SomeEvent struct {
   message.ObjectMeta //all types must have embedded ObjectMeta
   MyID      string    `json:"my_id"`
   HandledAt time.Time `json:"handled_at"`
}

type Handler struct {
}

func (h Handler) handleSomeCommand(execCtx execution.MessageExecutionCtx) error {
   someCmd, _ := execCtx.Message().Payload().(*SomeCommand)

   execCtx.Logger().Logf(log.InfoLevel, "Just received and handled command with MyID %s", someCmd.MyID)

   return execCtx.Send(message.NewOutcomingMessage(&SomeEvent{MyID: someCmd.MyID, HandledAt: time.Now()})) //reply with an event
}

func (h Handler) handleSomeEvent(execCtx execution.MessageExecutionCtx) error {
   ev, _ := execCtx.Message().Payload().(*SomeEvent)

   execCtx.Logger().Logf(log.InfoLevel, "Received event that was a response to a handled command %s at %s", ev.MyID, ev.HandledAt)

   return nil
}
```

Full working example is available [here](https://github.com/go-foreman/foreman-examples/tree/master/cmd/simple)

How to run it?

`docker compose up` - in the root directory

`cd cmd/simple && go build`

`./simple` - run the simple app

Some of the execution logs

![Untitled](Getting%20started/screenshot-logs.png)