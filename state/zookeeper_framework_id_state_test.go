package state

import (
	"fmt"

	"github.com/reverb/go-utils/rvb_zk"

	"code.google.com/p/goprotobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/reverb/go-mesos/mesos"
	"github.com/samuel/go-zookeeper/zk"
)

var _ = Describe("FrameworkIDState", func() {
	var (
		zkCluster *zk.TestCluster
		curator   *rvb_zk.Curator
	)

	BeforeEach(func() {
		cl, err := zk.StartTestCluster(1)
		if err != nil {
			Fail(fmt.Sprintf("%v", err))
		}
		zkCluster = cl

		c, err := zkCluster.ConnectAll()
		if err != nil {
			Fail(fmt.Sprintf("%v", err))
		}
		curator = rvb_zk.NewCurator(c, "")
	})

	AfterEach(func() {
		curator.Close()
		zkCluster.Stop()
	})

	It("should initialize a client", func() {
		path := "/golangstate/test-133"
		data := "originalId"
		pb, _ := proto.Marshal(&mesos.FrameworkID{Value: &data})
		curator.CreatePathRecursively(path, pb, 0, zk.WorldACL(zk.PermAll))

		cache := NewZookeeperFrameworkIDState(path, curator)
		cache.Start(false)
		defer cache.Stop()

		Ω(cache.Get().GetValue()).Should(Equal(data))
	})

	It("should set the value", func() {
		path := "/golangstate/test-2"
		data := "thefwid"
		cache := NewZookeeperFrameworkIDState(path, curator)
		cache.Start(false)
		defer cache.Stop()

		expected := &mesos.FrameworkID{Value: &data}
		cache.Set(expected)
		Ω(cache.Get()).Should(Equal(expected))
		d, _, _ := curator.Get(path)
		pb := &mesos.FrameworkID{}
		proto.Unmarshal(d, pb)
		Ω(pb).Should(Equal(expected))
	})

})
