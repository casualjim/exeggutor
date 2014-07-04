package state

import (
	"fmt"
	"testing"
	"time"

	"code.google.com/p/goprotobuf/proto"
	"github.com/reverb/go-mesos/mesos"
	"github.com/reverb/go-utils/rvb_zk"
	"github.com/samuel/go-zookeeper/zk"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFrameworkIDState(t *testing.T) {

	Convey("FrameworkIDState", t, func() {
		zkCluster, err := zk.StartTestCluster(1)
		So(err, ShouldBeNil)

		c, err := zkCluster.ConnectAll()
		So(err, ShouldBeNil)
		curator := rvb_zk.NewCurator(c, fmt.Sprintf("%d", time.Now().UnixNano()))

		Reset(func() {
			curator.Close()
			zkCluster.Stop()
		})

		Convey("should initialize a client", func() {
			path := "/golangstate/test-133"
			data := "originalId"
			pb, _ := proto.Marshal(&mesos.FrameworkID{Value: &data})
			curator.CreatePathRecursively(path, pb, 0, zk.WorldACL(zk.PermAll))

			cache := NewZookeeperFrameworkIDState(path, curator)
			cache.Start(false)
			defer cache.Stop()

			So(cache.Get().GetValue(), ShouldResemble, data)
		})

		Convey("should set the value", func() {
			path := "/golangstate/test-2"
			data := "thefwid"
			cache := NewZookeeperFrameworkIDState(path, curator)
			cache.Start(false)
			defer cache.Stop()

			expected := &mesos.FrameworkID{Value: &data}
			cache.Set(expected)
			So(cache.Get(), ShouldResemble, expected)
			d, _, _ := curator.Get(path)
			pb := &mesos.FrameworkID{}
			proto.Unmarshal(d, pb)
			So(pb, ShouldResemble, expected)
		})

	})
}
