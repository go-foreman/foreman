# AMQP Reconnect — Consume Not Restarting After Reconnect

## Symptom
After the AMQP connection drops and reconnects, `Consume` is never restarted. Messages pile up in the queue as `Ready` even though the connection has been reestablished.

Files involved:
- `pubsub/transport/amqp/amqp.go`
- `pubsub/transport/amqp/connection.go`
- `pubsub/subscriber/subscriber.go` (affected downstream)

## Root cause candidates

### 1. `reconnectedCount` never resets on successful `Consume`
`connection.go:175-208` — `Channel.Consume`:

```go
var reconnectedCount uint  // declared once, never reset
...
for {
    d, err := ch.AmqpChannel.Consume(...)
    if err != nil {
        if reconnectedCount > reconnectCount { break }  // terminal
        reconnectedCount++
        continue
    }
    // success — but reconnectedCount stays
    for msg := range d { deliveries <- msg }
    ...
}
```

After a handful of intermittent reconnect hiccups (20 cumulative errors over the consumer's lifetime), the loop breaks forever, `deliveries` closes, `subscriber.Run` exits at `subscriber.go:131-134`, and no one re-registers consumers. Queue stays `Ready`.

### 2. Two competing reconnect paths that race each other with no coordination
- `Connection.Channel` (`connection.go:107-143`) has a goroutine listening on `channel.NotifyClose` that swaps `channel.AmqpChannel = newCh`.
- `Channel.Consume` (`connection.go:177-209`) has its own retry loop calling `ch.AmqpChannel.Consume(...)`.

They race, the swap is unsynchronized (data race on the `AmqpChannel` field), and the retry loop has no way of knowing when the new channel is ready — it just blind-retries and hopes the field was updated between sleeps.

### 3. `Qos` not re-applied after channel swap
`amqp.go:202-206` sets `Qos` once at `Consume()` setup. When `Connection.Channel`'s reconnect goroutine replaces `channel.AmqpChannel`, the new raw channel has no prefetch cap.

### 4. NotifyClose edge case in `Connection.Channel`
`connection.go:111` — if the NEW channel is already closed (or closes very quickly) when we call `channel.NotifyClose(...)` on it after a swap, amqp091 closes the passed chan immediately → `!ok` is true → the reconnect goroutine exits permanently → `AmqpChannel` is never swapped again → `Channel.Consume`'s retry loop burns through its 20 attempts on the stale channel and dies.

### 5. `Dial` hard-exits the process after 20 reconnects
`connection.go:50` — `logger.Logf(log.FatalLevel, ...)` with the default logger calls `os.Exit(1)`. If a RabbitMQ outage exceeds ~60s (20 × 3s), the whole process dies. Doesn't match the "connection reestablished" symptom but worth fixing while we're in the file.

## Proposed fixes

### Option A — Collapse the two reconnect mechanisms (recommended)
Drop `Connection.Channel`'s swap-on-notify goroutine. Make `Channel.Consume`'s loop re-acquire a fresh `*amqp.Channel` from `Connection` on each retry and re-apply `Qos`. This removes the race, removes the stale-field problem, and `reconnectedCount` can then be reset on each successful `Consume` call.

Requires `Channel` to hold a reference back to its owning `Connection` so it can ask for a new channel.

### Option B — Surgical patch (smaller blast radius)
Keep the current structure but fix the concrete bugs:
- Reset `reconnectedCount = 0` after each successful `Consume` call in `Channel.Consume`.
- Re-apply `Qos` after the `AmqpChannel` swap in `Connection.Channel` (store the Qos settings on `Channel`).
- Guard the `AmqpChannel` field with a mutex (or atomic.Value) to remove the data race.
- Handle the `!ok` case in the `Connection.Channel` reconnect goroutine by re-entering the reconnect inner loop instead of exiting.
- Remove the `FatalLevel` call in `Dial` so a long outage doesn't kill the process.

## Open question
Which option to go with? Option A is cleaner and eliminates a class of bugs; Option B is a minimal change but leaves the dual-reconnect design in place.