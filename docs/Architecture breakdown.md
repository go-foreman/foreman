# Architecture breakdown

BPMN diagram

## Components:

PubSub implementation consists of 5 main components: `transport`, `subscriber`, `dispatcher`, `message` and `endpoint`. 

---

### Transport

Transport is an abstraction over messaging protocol that knows how to create a queue, topic as well as  consume and send packages. 

```go
type Transport interface {
// CreateTopic creates a topic(exchange) in message broker
	CreateTopic(ctx context.Context, topic Topic) error
	// CreateQueue creates a queue in a message broker
	CreateQueue(ctx context.Context, queue Queue, queueBind ...QueueBind) error
	// Consume starts receiving packages in a gorotuine and sends them to the <-chan IncomingPkg
	Consume(ctx context.Context, queues []Queue, options ...ConsumeOpts) (<-chan IncomingPkg, error)
	// Send sends an outbound package to a defined destination topic in OutboundPkg
	Send(ctx context.Context, outboundPkg OutboundPkg, options ...SendOpts) error
	// Connect connects to a message broker. It should be able to reconnect automatically in case of failure. 
	Connect(context.Context) error
	// Disconnect disconnects from a message broker and stops listening for packages.
	Disconnect(context.Context) error
}
```

Each package has metadata, payload and knows how to send acknowledgment or reject it. 

```go
type IncomingPkg interface {
   UID() string
   Origin() string
   Payload() []byte
   Headers() map[string]interface{}
   Ack(options ...AcknowledgmentOption) error
   Nack(options ...AcknowledgmentOption) error
   Reject(options ...AcknowledgmentOption) error
   ReceivedAt() time.Time
   PublishedAt() time.Time
}
```

Only AMQP implementation is currently available, Apache Kafka is defined in the roadmap.  Transport is used in `subscriber` and `endpoint` packages which consume and send packages accordingly.  `Connect()` must be called by user explicitly, usually before creating topics and queues.

---

### Subscriber

This package is a main running process of the Foreman. Interface is fairly simple:

```go
type Subscriber interface {
   // Run listens queues for packages and processes them. Gracefully shuts down either on os.Signal or ctx.Done()
	Run(ctx context.Context, queues ...transport.Queue) error
}
```

In default implementation `Run` method uses transport to consume packages from queues, then schedules one of concurrent workers from a pool to work on a received package. Each worker is a goroutine that is managed by pool's dispatcher. `Subscriber` blocks and waits for a free worker If all of them are busy. 

<aside>
💡 Interesting moment here: a worker waits for a message, not backwards. There could be a case when the message received, but no workers were available till subscriber is stopped.  So this message couldn't have been processed, but was received, not processed and not acknowledged.

</aside>

Each worker has configurable time for waiting, once time passed, the worker returns to the pool.  This flow was needed to avoid keeping loop blocked and allow listening for `context.Done()` or `os.Signal`.  

The worker processes the package with `Processor` .  The last one uses `Marshaller` to decode a received package, matches it’s type with a list of `Executor`'s  in `Dispatcher` and calls them in order on the package. 

Acknowledgement is sent once `Processor` had finished without errors. The worker signals that he is free to work again.  

```go
type Processor interface {
   Process(ctx context.Context, inPkg transport.IncomingPkg) error
}
```

---

### Dispatcher

Dispatcher allows to register a message type to a list of  `Executor` 's and match the type with the list. 

```go
type Dispatcher interface {
   // Match matches object type and returns list of registered executors for this type
	Match(obj message.Object) []execution.Executor
	// SubscribeForCmd subscribes given executor for a command
	SubscribeForCmd(obj message.Object, executor execution.Executor) Dispatcher
	// SubscribeForEvent subscribes given executor for an event
	SubscribeForEvent(obj message.Object, executor execution.Executor) Dispatcher
	// SubscribeForAllEvents subscribes executor type for all types
	SubscribeForAllEvents(executor execution.Executor) Dispatcher
}
```

Subscription for commands and events is separated to differentiate types of executors and be more explicit when defining a command handler or an event listener.

 `SubscribeForAllEvents` subscribes for all event types on which were previously subscribed with `SubscribeForEvent`

API of `Dispatcher` allows chaining of methods when subscribing. 

---

### Scheme

Before going to our next component we need to understand what is `scheme` .

It's a registry of types (`reflect.Type)` and their metadata `GroupKind` which is used to unmarshall a dynamic message by first reading the metadata and then mapping payload to the matched type.  All types can be registered in `init()` function as an easy option.

```go
type KnownTypesRegistry interface {
   // AddKnownTypes registers list of types of objects to a Group. Kind of each type will be set as struct name using reflection
   AddKnownTypes(gv Group, types ...Object)
   // AddKnownTypeWithName registers a type an object to a Group and custom defined Kind
   AddKnownTypeWithName(gk GroupKind, obj Object)
   // NewObject instantiates new object instance of a type registered behind GroupKind
   NewObject(gk GroupKind) (Object, error)
   // ObjectKind returns GroupKind of an already registered type
   ObjectKind(object Object) (*GroupKind, error)
}
```

### Message

This package contains three main units: `Object`, `Marshaller` and `MessageExecutionCtx`

`Object` is an interface which must be implemented by each message type in the system in order to be sent or received.  It wraps `scheme.Object` interface. 

```go
// Object interface must be supported by all message types registered with Scheme. Since objects in a scheme are
// expected to be marshalled to the wire, the interface an Object must provide to the Scheme allows
// marshallers to set the kind, and group the object is represented as
type Object interface {
   GroupKind() GroupKind
   SetGroupKind(gk *GroupKind)
}
```

A type can implement this interface just by embedding `scheme.ObjectMeta` struct. 

`Marshaller` knows how to marshal and unmarshal message. 

```go
type Marshaller interface {
   // Unmarshal decodes received bytes into original type that must be registered in scheme
   Unmarshal(b []byte) (Object, error)
   // Marshal encodes a type and automatically adds needed metadata to resulting bytes
   Marshal(obj Object) ([]byte, error)
}
```

`MessageExecutionCtx` is an execution context of each message. It's passed to handler as a single param.  

 

```go
// MessageExecutionCtx is passed to each executor and contains received message, ctx, knows how to send out or return a message.
type MessageExecutionCtx interface {
	// Message returns received message
	Message() *message.ReceivedMessage
	// Context returns parent execution context. Each message has own time limit in which it must be processed.
	Context() context.Context
	// Send sends an out coming message to registered endpoints
	Send(message *message.OutcomingMessage, options ...endpoint.DeliveryOption) error
	// Return sends received message to registered endpoints and updates number of returns in headers
	Return(options ...endpoint.DeliveryOption) error
	// LogMessage allows to log message in handlers
	LogMessage(level log.Level, msg string)
}

// Executor is a callback that will be called on received message with context.
// it should return an error only if internal server error happened, in case of business errors it's best to send a failure event.  
type Executor func(execCtx MessageExecutionCtx) error
```

Any `Executor`  can access received message from the context, reply with another message after processing or return it back. 

### Endpoint

Endpoint is an end place to which messages are being sent. 

```go
// Endpoint knows where to deliver a message
type Endpoint interface {
	// Name is a unique name of the endpoint
	Name() string
	// Send sends a message with specified implementation
	Send(ctx context.Context, message *message.OutcomingMessage, options ...DeliveryOption) error
}
```

It's possible to register a single message type for multiple endpoints.  

```go
// Router is a registry of Endpoints and types. Each type can have multiple endpoints assigned.
type Router interface {
   // RegisterEndpoint assigns types of objects to an endpoint
   RegisterEndpoint(endpoint Endpoint, objects ...message.Object)
   // Route returns a list of endpoints that were assigned to a type of object
   Route(obj message.Object) []Endpoint
}
```