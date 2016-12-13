package hub

import (
	"sync"
	"github.com/DeanThompson/syncmap"
	"github.com/hectane/go-nonblockingchan"
	"github.com/bixi/linked/llist"
	"github.com/bixi/linked/node"
)

const (
	publishCommand = 0
	addSubCommand = 1
	removeSubCommand = 2
	stopCommand = 3
)

type brokerCommand struct {
	node.Dnode
	commandType uint
}

type localBroker struct {
	subscribers   *llist.List
	commandList   *llist.List
	bgCommandList *llist.List
	commandChan   chan bool
	disposed      bool
	pubsub        *localPubSub
	key           string
	disposedLock  sync.RWMutex
	sync.Mutex
}

func newLocalBroker(pubsub *localPubSub, key string) *localBroker {
	broker := new(localBroker)
	broker.pubsub = pubsub
	broker.key = key
	broker.subscribers = &llist.List{}
	broker.commandList = &llist.List{}
	broker.bgCommandList = &llist.List{}
	broker.commandChan = make(chan bool)
	go func() {
		for {
			for {
				broker.Lock()
				broker.bgCommandList.Swap(broker.commandList)
				broker.Unlock()
				if (broker.bgCommandList.Length() == 0) {
					<-broker.commandChan
				} else {
					break;
				}
			}

			command := broker.bgCommandList.Head().(*brokerCommand)
			for command != nil {
				switch command.commandType {
				case publishCommand:
					s := broker.subscribers.Head()
					for s != nil {
						message, _ := command.Data()
						s.(*localSubscriber).channel.Send <- message
						s, _, _ = s.Next()
					}
					break;
				case addSubCommand:
					s, _ := command.Data()
					broker.subscribers.Append(s.(*localSubscriber))
					break;
				case removeSubCommand:
					s, _ := command.Data()
					sub := s.(*localSubscriber)
					broker.subscribers.Remove(sub)
					close(sub.channel.Send)
					if (broker.subscribers.Length() == 0) {
						broker.disposedLock.Lock()
						broker.disposed = true
						broker.pubsub.remove(broker.key)
						broker.disposedLock.Unlock()
						return
					}
					break;
				case stopCommand:
					data, _ := command.Data()
					c := data.(chan bool)
					broker.Lock()
					defer broker.Unlock()
					s := broker.subscribers.Head()
					for s != nil {
						sub := s.(*localSubscriber)
						close(sub.channel.Send)
						s, _, _ = s.Next()
					}
					broker.subscribers.Clear()
					broker.disposed = true
					c <- true
					return
					break;
				}
				c, _, _ := command.Next()
				if (c == nil) {
					command = nil
				} else {
					command = c.(*brokerCommand)
				}
			}
			broker.bgCommandList.Clear()
		}
	}()
	return broker
}

func (broker *localBroker) sendCommand(command *brokerCommand) {
	broker.Lock()
	broker.commandList.Append(command)
	broker.Unlock()
	select {
	case broker.commandChan <- true:
		break;
	default:
		break;
	}
}

func (broker *localBroker) publish(message interface{}) {
	command := &brokerCommand{}
	command.commandType = publishCommand
	command.SetData(message)
	broker.sendCommand(command)
}

func (broker *localBroker) subscribe() Subscriber {
	subscriber := newLocalSubscriber()
	broker.addSubscriber(subscriber)
	return subscriber
}

func (broker *localBroker) addSubscriber(subscriber*localSubscriber) {
	if (subscriber.broker == nil) {
		subscriber.broker = broker

		command := &brokerCommand{}
		command.commandType = addSubCommand
		command.SetData(subscriber)
		broker.sendCommand(command)
	}
}

func (broker *localBroker) removeSubscriber(subscriber *localSubscriber) {
	if (subscriber.broker == broker) {
		subscriber.broker = nil
		command := &brokerCommand{}
		command.commandType = removeSubCommand
		command.SetData(subscriber)
		broker.sendCommand(command)
	}
}

func (broker *localBroker) stop(done chan bool) <-chan bool {
	command := &brokerCommand{}
	command.commandType = stopCommand
	command.SetData(done)
	broker.sendCommand(command)
	return done
}

type localSubscriber struct {
	node.Dnode
	broker  *localBroker
	channel *nbc.NonBlockingChan
	sync.RWMutex
}

func newLocalSubscriber() *localSubscriber {
	subscriber := &localSubscriber{}
	subscriber.channel = nbc.New()
	return subscriber
}

func (subscriber *localSubscriber) Receive() <-chan interface{} {
	return subscriber.channel.Recv
}

func (subscriber *localSubscriber) Close() {
	if (subscriber.broker != nil) {
		subscriber.broker.removeSubscriber(subscriber)
	}
}

type localSubject struct {
	key    string
	pubsub *localPubSub
	broker *localBroker
}

func (ls *localSubject) Publish(message interface{}) {
	ls.broker.disposedLock.RLock()
	if (ls.broker.disposed) {
		ls.broker.disposedLock.RUnlock()
		ls.broker = ls.pubsub.broker(ls.key)
		ls.Publish(message)
	} else {
		ls.broker.publish(message)
		ls.broker.disposedLock.RUnlock()
	}
}

func (ls *localSubject) Subscribe() Subscriber {
	ls.broker.disposedLock.RLock()
	if (ls.broker.disposed) {
		ls.broker.disposedLock.RUnlock()
		ls.broker = ls.pubsub.broker(ls.key)
		return ls.Subscribe()
	} else {
		defer ls.broker.disposedLock.RUnlock()
		return ls.broker.subscribe()
	}
}

type localPubSub struct {
	brokers *syncmap.SyncMap
	sync.Mutex
}

func NewLocalPubSub() PubSub {
	var lps localPubSub
	lps.brokers = syncmap.New()
	return &lps
}

func (lps *localPubSub) Subject(key string) Subject {
	ls := &localSubject{}
	ls.broker = lps.broker(key)
	ls.pubsub = lps
	return ls
}

func (lps *localPubSub) Stop() <- chan bool {
	lps.Lock()
	defer lps.Unlock()
	iter := lps.brokers.IterItems()
	c := make(chan bool)
	done := make(chan bool)
	count := 0
	for item := range iter {
		broker := item.Value.(*localBroker)
		broker.stop(c)
		count++
	}
	lps.brokers.Flush()
	go func() {
		for i := 0; i < count; i++ {
			<-c
		}
		done <- true
	}()
	return done
}

func (lps *localPubSub) remove(key string) {
	lps.brokers.Delete(key)
}

func (lps *localPubSub) broker(key string) *localBroker {
	if s, ok := lps.brokers.Get(key); ok {
		return s.(*localBroker)
	} else {
		lps.Lock()
		defer lps.Unlock()
		if s, ok = lps.brokers.Get(key); ok {
			return s.(*localBroker)
		}
		s = newLocalBroker(lps, key)
		lps.brokers.Set(key, s)
		return s.(*localBroker)
	}
}
