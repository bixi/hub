package hub

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEmptyStopper(t *testing.T) {
	Convey("Given an EmptyStopper", t, func() {
		stopper := &EmptyStopper{}
		Convey("When call Stop", func() {
			c := stopper.Stop()
			Convey("Then chan should return immediatly", func() {
				b := <- c
				So(b, ShouldEqual, false)
			})
		})
	})
}
