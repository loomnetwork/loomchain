# go-pubsub

Simple publish subscribe queue. This is available only inside application.
There is no intent to make this span across processes/machines.

## Examples

```go

ps := pubsub.New()
subscriber := ps.Subscribe("user:1", "user:2").Do(func(message Message) {
    println("We got meessage on topic", message.Topic())
})

defer subscriber.Close()

count := ps.Publish(NewMessage("user:1:username", "hello"))

fmt.Printf("Published messages to %v subscribers", count)

```

go-pubsub has also default Hub, if you don't need to track your own Hub.

```go
subscriber := pubsub.Subscribe("user:1", "user:2").Do(func(message Message) {
    println("We got meessage on topic", message.Topic())
})

defer subscriber.Close()

count := pubsub.Publish(NewMessage("user:1:username", "hello"))

fmt.Printf("Published messages to %v subscribers", count)

```

# tests

If you want to run tests, you need goconvey to be installed. You can install it by typing:

    $ go get github.com/smartystreets/goconvey

and then you can run

    go test

or you can run goconvey for more info, navigate into go-pubsub directory and run:

    goconvey

# author
Peter Vrba <phonkee@phonkee.eu>