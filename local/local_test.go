package local

import (
	"fmt"
	"testing"

	. "github.com/bixi/hub"
	"github.com/bradfitz/iter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestLocalPubSub(t *testing.T) {
	Convey("Given a localPubSub", t, func() {
		ps := NewPubSub()
		Convey("Given a Subject 'Hello'", func() {
			s := ps.Subject("Hello")
			Convey("When subscribe to the subject", func() {
				sub := s.Subscribe()
				Convey("Then subscriber should receive message from subject", func() {
					s.Publish("World")
					v := <-sub.Receive()
					So(v, ShouldEqual, "World")
				})
				Convey("Then subscriber should receive nil after close", func() {
					sub.Close()
					s.Publish("World")
					v := <-sub.Receive()
					So(v, ShouldEqual, nil)
				})
			})
			Convey("When three subscribers subs to the same subject", func() {
				sub1 := s.Subscribe()
				sub2 := s.Subscribe()
				sub3 := s.Subscribe()
				Convey("Then they should receive the same message from subject", func() {
					s.Publish("Meteor")
					v1 := <-sub1.Receive()
					v2 := <-sub2.Receive()
					v3 := <-sub3.Receive()
					So(v1, ShouldEqual, "Meteor")
					So(v1, ShouldEqual, v2)
					So(v1, ShouldEqual, v3)
				})
				Convey("Then they should receive nil after close", func() {
					sub1.Close()
					sub3.Close()
					s.Publish("Comet")
					v1 := <-sub1.Receive()
					v2 := <-sub2.Receive()
					v3 := <-sub3.Receive()
					So(v1, ShouldEqual, nil)
					So(v2, ShouldEqual, "Comet")
					So(v1, ShouldEqual, v3)
					sub2.Close()
				})
			})
			Convey("Then get Subject 'Hello' should return the same data", func() {
				s1 := ps.Subject("Hello").(*localSubject)
				s2 := ps.Subject("Hello").(*localSubject)
				So(*s2 == *s1, ShouldBeTrue)
				c := make(chan Subject)
				f := func() {
					c <- ps.Subject("Planet")
				}
				for range iter.N(10) {
					go f()
				}
				planet := ps.Subject("Planet").(*localSubject)
				for range iter.N(10) {
					planet1 := (<-c).(*localSubject)
					So(planet.key, ShouldEqual, planet1.key)
					So(planet.pubsub, ShouldEqual, planet1.pubsub)
					if !planet.broker.disposed && !planet1.broker.disposed {
						So(planet.broker, ShouldEqual, planet1.broker)
					}
				}
			})
			Convey("Then get Subject 'Other' should return the other object", func() {
				s1 := ps.Subject("Other").(*localSubject)
				So(*s.(*localSubject) != *s1, ShouldBeTrue)
			})
		})
	})
	Convey("Given a localPubSub", t, func() {
		ps := NewPubSub()
		Convey("When lots of publisher and subscribers done.", func() {
			c1 := make(chan bool)
			c2 := make(chan bool)
			pubFunc := func(count int) {
				s := ps.Subject("Ants")
				s.Publish(count)
				c1 <- true
			}
			subFunc := func(count int) {
				s := ps.Subject("Ants")
				sub := s.Subscribe()
				sub.Close()
				c2 <- true
			}
			for i := range iter.N(100) {
				go subFunc(i)
				go pubFunc(i)
			}
			for range iter.N(100) {
				<-c1
				<-c2
			}
			Convey("Then the pubsub should still running normally.", func() {
				s := ps.Subject("Ants")
				sub := s.Subscribe()
				s.Publish("Hello")
				m := <-sub.Receive()
				sub.Close()
				So(m, ShouldEqual, "Hello")
			})
		})
		Convey("When lots of publisher and subscribers running.", func() {
			c := make(chan bool)
			pubFunc := func(count int) {
				s := ps.Subject(fmt.Sprintf("Ants%v", count%20))
				s.Publish(count)
				c <- true
			}
			subFunc := func(count int) {
				s := ps.Subject(fmt.Sprintf("Ants%v", count%20))
				sub := s.Subscribe()
				c <- true
				for {
					<-sub.Receive()
				}
			}
			for i := range iter.N(200) {
				go subFunc(i)
				go pubFunc(i)
			}
			for range iter.N(400) {
				<-c
			}
			Convey("Then the pubsub should contain no broker after stopped.", func() {
				done := ps.Stop()
				<-done
				So(ps.(*localPubSub).brokers.Size(), ShouldEqual, 0)
			})
		})
	})
}
