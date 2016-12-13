package hub

type Subject interface {
	Publish(message interface{})
	Subscribe() Subscriber
}

type Subscriber interface {
	Receive() <-chan interface{}
	Close()
}

type Stopper interface {
	Stop() <- chan bool
}

type PubSub interface {
	Stopper
	Subject(key string) Subject
}

// ------------------------------

type EmptyStopper struct {

}

func (s *EmptyStopper) Stop() <- chan bool {
	c := make(chan bool)
	close(c)
	return c
}

