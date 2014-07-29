// Package sla provides ...
package sla

import (
	stdlog "log"
	"os"
	"testing"
	"time"

	"github.com/op/go-logging"
	"github.com/reverb/exeggutor/store"
	app_store "github.com/reverb/exeggutor/store/apps"
	task_store "github.com/reverb/exeggutor/store/tasks"
	task_queue "github.com/reverb/exeggutor/tasks/queue"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSLAMonitor(t *testing.T) {

	logBackend := logging.NewLogBackend(os.Stderr, "", stdlog.LstdFlags|stdlog.Lshortfile)
	logBackend.Color = true
	logging.SetBackend(logBackend)
	logging.SetLevel(logging.ERROR, "")

	Convey("A SLA Monitor", t, func() {
		q := &task_queue.PrioQueue{}
		tq := task_queue.NewTaskQueueWithPrioQueue(q)
		tq.Start()
		as := store.NewEmptyInMemoryStore()
		ts := store.NewEmptyInMemoryStore()

		monitor := &simpleSLAMonitor{
			taskStore: task_store.NewWithStore(ts),
			appStore:  app_store.NewWithStore(as),
			queue:     tq,
			interval:  1 * time.Second,
		}
		monitor.Start()

		Reset(func() {
			tq.Stop()
			monitor.Stop()
		})

		Convey("", func() {})

	})
}
