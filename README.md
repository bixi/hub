# hub
A high performance, non-blocking async Publish/Subscribe pattern abstraction.

## Usage
### install
```shell
$ go get github.com/bixi/hub
```

### example
```go
pubsub := hub.NewLocalPubSub()
subject := pubsub.Subject("Hello")

// publish
subject.Publish("Aloha")

// subscribe (usually in another goroutine)
subscriber := subject.Subscribe()
for {
    message := <- subscriber.Receive()
    fmt.Printf("%v\n", message)
}
```

## hub types
Local Hub: In-memory implementation of pub/sub, for the purpose of decoupling goroutines, while keeping the choice of switching to message broker middlewares(NATS, Redis, etc.) in the future.

### TODO
* NATS support.
* Redis support.
* RPC support.