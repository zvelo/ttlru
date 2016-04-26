package ttlru

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTTL(t *testing.T) {
	Convey("TestTTL", t, func() {
		Convey("General functionality", func() {
			l := New(128, 2*time.Second)
			So(l, ShouldNotBeNil)
			So(l.Len(), ShouldEqual, 0)
			So(l.Cap(), ShouldEqual, 128)

			for i := 0; i < 128; i++ {
				So(l.Set(i, i), ShouldBeFalse)
			}

			So(l.Len(), ShouldEqual, 128)
			So(l.Cap(), ShouldEqual, 128)

			for i := 128; i < 256; i++ {
				So(l.Set(i, i), ShouldBeTrue)
			}

			So(l.Len(), ShouldEqual, 128)
			So(l.Cap(), ShouldEqual, 128)

			for _, k := range l.Keys() {
				v, ok := l.Get(k)
				So(ok, ShouldBeTrue)
				So(v, ShouldEqual, k)
			}

			for i := 0; i < 128; i++ {
				val, ok := l.Get(i)
				So(ok, ShouldBeFalse)
				So(val, ShouldBeNil)
			}

			for i := 128; i < 256; i++ {
				val, ok := l.Get(i)
				So(ok, ShouldBeTrue)
				So(val, ShouldEqual, i)
			}

			for i := 128; i < 192; i++ {
				So(l.Del(i), ShouldBeTrue)
				val, ok := l.Get(i)
				So(ok, ShouldBeFalse)
				So(val, ShouldBeNil)
			}

			done := make(chan interface{})

			time.AfterFunc(3*time.Second, func() {
				Convey("TTL Works", t, func() {
					So(l.Len(), ShouldEqual, 0)
					So(l.Cap(), ShouldEqual, 128)

					So(l.Set(0, 0), ShouldBeFalse)
					So(l.Len(), ShouldEqual, 1)
					So(l.Cap(), ShouldEqual, 128)

					l.Purge()
					So(l.Len(), ShouldEqual, 0)
					So(l.Cap(), ShouldEqual, 128)

					val, ok := l.Get(200)
					So(ok, ShouldBeFalse)
					So(val, ShouldBeNil)

					done <- true
				})
			})

			<-done
		})

		Convey("Add returns properly", func() {
			l := New(1, 2*time.Second)
			So(l, ShouldNotBeNil)
			So(l.Len(), ShouldEqual, 0)
			So(l.Cap(), ShouldEqual, 1)

			So(l.Set(1, 1), ShouldBeFalse)
			So(l.Len(), ShouldEqual, 1)
			So(l.Cap(), ShouldEqual, 1)

			So(l.Set(2, 2), ShouldBeTrue)
			So(l.Len(), ShouldEqual, 1)
			So(l.Cap(), ShouldEqual, 1)
		})

		Convey("Invalid creation", func() {
			So(New(0, 1), ShouldBeNil)
			So(New(1, 0), ShouldBeNil)
			So(New(-1, 1), ShouldBeNil)
			So(New(1, -1), ShouldBeNil)
		})

		Convey("Set should also update", func() {
			l := New(1, 2*time.Second)
			So(l, ShouldNotBeNil)
			So(l.Len(), ShouldEqual, 0)
			So(l.Cap(), ShouldEqual, 1)

			So(l.Set(1, 1), ShouldBeFalse)
			So(l.Len(), ShouldEqual, 1)
			So(l.Cap(), ShouldEqual, 1)
			v, ok := l.Get(1)
			So(ok, ShouldBeTrue)
			So(v, ShouldEqual, 1)

			So(l.Set(1, 2), ShouldBeFalse)
			So(l.Len(), ShouldEqual, 1)
			So(l.Cap(), ShouldEqual, 1)
			v, ok = l.Get(1)
			So(ok, ShouldBeTrue)
			So(v, ShouldEqual, 2)
		})

		Convey("Delete should return properly", func() {
			l := New(1, 2*time.Second)
			So(l, ShouldNotBeNil)
			So(l.Len(), ShouldEqual, 0)
			So(l.Cap(), ShouldEqual, 1)

			So(l.Set(1, 1), ShouldBeFalse)
			So(l.Len(), ShouldEqual, 1)
			So(l.Cap(), ShouldEqual, 1)
			v, ok := l.Get(1)
			So(ok, ShouldBeTrue)
			So(v, ShouldEqual, 1)

			So(l.Del(1), ShouldBeTrue)
			So(l.Del(2), ShouldBeFalse)
		})

		Convey("Item should expired despite Get()", func() {
			l := New(1, 300*time.Millisecond)
			So(l, ShouldNotBeNil)
			So(l.Set(1, 1), ShouldBeFalse)
			l.NoReset = true

			done := make(chan interface{})
			time.AfterFunc(200*time.Millisecond, func() {
				Convey("Get() item works", t, func() {
					v, ok := l.Get(1)
					So(ok, ShouldBeTrue)
					So(v, ShouldEqual, 1)
					done <- true
				})
			})
			<-done

			time.AfterFunc(200*time.Millisecond, func() {
				Convey("Item is gone despite the Get()", t, func() {
					v, ok := l.Get(1)
					So(ok, ShouldBeFalse)
					So(v, ShouldBeNil)
					done <- true
				})
			})
			<-done

		})
	})
}
