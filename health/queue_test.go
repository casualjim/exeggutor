package health

import (
	stdlog "log"
	"os"
	"testing"
	"time"

	// . "github.com/reverb/go-utils/convey/matchers"

	"github.com/op/go-logging"
	"github.com/reverb/exeggutor/health/check"
	"github.com/reverb/exeggutor/protocol"
	. "github.com/smartystreets/goconvey/convey"
)

type mockHealthCheck struct {
	ID     string
	Result check.Result
}

func (m *mockHealthCheck) GetID() string {
	return m.ID
}

func (m *mockHealthCheck) Check() check.Result {
	return m.Result
}

func (m *mockHealthCheck) Update(config *protocol.HealthCheck) {
}

func (m *mockHealthCheck) Cancel() {
}

func makeActiveHealthCheck(id string, expiresAt time.Time) *activeHealthCheck {
	return &activeHealthCheck{
		ExpiresAt: expiresAt,
		HealthCheck: &mockHealthCheck{
			ID: id,
			Result: check.Result{
				ID:        id,
				Code:      protocol.HealthCheckResultCode_HEALTHY,
				NextCheck: expiresAt.Add(1 * time.Minute),
			},
		},
	}
}

// IPC: One might say this needs tests for thread-safety.
//      I think it's too hard (NP hard) to correct tests for thread-safety
//      So instead of lulling ourselves into a false sense of security
//      I'd rather leave it as wasted effort.
//      Each write operation is guarded with a mutex, which in the case of a trivial
//      class like this should be all we need to guarantee thread safety.

func TestHealthCheckQueue(t *testing.T) {

	logBackend := logging.NewLogBackend(os.Stderr, "", stdlog.LstdFlags|stdlog.Lshortfile)
	logBackend.Color = true
	logging.SetBackend(logBackend)
	logging.SetLevel(logging.ERROR, "")

	Convey("An ActiveHealthCheckQueue", t, func() {

		queue := newHealthCheckQueue()
		queue.Push(makeActiveHealthCheck("app-1", time.Now()))
		queue.Push(makeActiveHealthCheck("app-2", time.Now().Add(30*time.Second)))
		queue.Push(makeActiveHealthCheck("app-3", time.Now().Add(40*time.Second)))
		queue.Push(makeActiveHealthCheck("app-4", time.Now().Add(50*time.Second)))

		Convey("should report the correct length", func() {
			So(queue.Len(), ShouldEqual, 4)
		})

		Convey("should remove an item from the queue if it exists", func() {
			So(queue.Contains("app-3"), ShouldBeTrue)
			queue.Remove("app-3")
			So(queue.Contains("app-3"), ShouldBeFalse)
		})

		Convey("should return true when the queue contains an item", func() {
			So(queue.Contains("app-9394"), ShouldBeFalse)
			So(queue.Contains("app-3"), ShouldBeTrue)
		})

		Convey("when enqueueing", func() {
			Convey("should add items to the queue", func() {
				queue.Push(makeActiveHealthCheck("app-5", time.Now().Add(10*time.Second)))
				So(queue.Len(), ShouldEqual, 5)
				So(queue.Contains("app-5"), ShouldBeTrue)
			})

			Convey("should add items at the right order in the queue", func() {
				ac := makeActiveHealthCheck("app-5", time.Now().Add(10*time.Second))
				queue.Push(ac)
				So(ac.index, ShouldEqual, 1)
			})
		})

		Convey("when dequeueing", func() {

			Convey("should return nil when the queue is empty", func() {
				q := newHealthCheckQueue()
				ac, _, found := q.Pop()
				So(ac, ShouldBeNil)
				So(found, ShouldBeFalse)
			})

			Convey("should return nil when none of the items is expired", func() {
				cur := time.Now().Add(5 * time.Minute)
				q := newHealthCheckQueue()
				hc := makeActiveHealthCheck("app-21", cur)
				q.Push(hc)
				ac, _, found := q.Pop()
				So(ac, ShouldBeNil)
				So(found, ShouldBeFalse)
				So(q.Len(), ShouldEqual, 1)
			})

			Convey("should return an item when it is expired", func() {
				cur := time.Now().Add(-1 * time.Minute)
				hc := makeActiveHealthCheck("app-22", cur)
				q := newHealthCheckQueue()
				q.Push(hc)
				ac, _, found := q.Pop()
				So(ac, ShouldEqual, hc)
				So(found, ShouldBeTrue)
				So(q.Len(), ShouldEqual, 0)
			})
		})

	})
}
