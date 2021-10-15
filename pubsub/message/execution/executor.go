package execution

// Executor is a callback that will be called on received message with context.
// it should return an error only if internal server error happened, in case of business errors it's best to send a failure event.
type Executor func(execCtx MessageExecutionCtx) error
