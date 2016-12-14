package hub

// Subject a pub/sub subject.
type Subject interface {
	Publish(message interface{})
	Subscribe() Subscriber
}

// Subscriber a subscriber to a subject.
// Calling Close() is necessary
// or it will lead to memory leak.
type Subscriber interface {
	Receive() <-chan interface{}
	Close()
}

// Stopper provides Stop() interface for PubSub
// to cleanup and flush suspending messages.
type Stopper interface {
	Stop() <-chan bool
}

// PubSub holds subjects,
// Publishers and Subscribers can communicate
// with each other via the same subject.
type PubSub interface {
	Stopper
	Subject(key string) Subject
}

// ------------------------------

// EmptyStopper provides an empty implementation of Stopper.
type EmptyStopper struct {
}

// Stop does nothing.
func (s *EmptyStopper) Stop() <-chan bool {
	c := make(chan bool)
	close(c)
	return c
}
