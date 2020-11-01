package execution

type Executor func(execCtx MessageExecutionCtx) error
