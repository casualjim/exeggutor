package api_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/reverb/exeggutor/agora/api"
	"github.com/reverb/exeggutor/store"
)

// func

var _ = Describe("ApplicationsApi", func() {

	var (
		data store.KVStore
	)

	BeforeEach(func() {
		data = store.NewEmptyInMemoryStore()
		data.Start()
	})

	AfterEach(func() {
		data.Stop()
	})

	Context("List all applications", func() {
		It("returns a 200 Status Code", func() {
			Mount(ApplicationsContext{AppStore: data}, (*ApplicationsContext).ListAll).Get("/")
			Expect(response.Code).To(Equal(200))
			Expect(response.Body.String()).To(Equal("[]"))
		})
	})

	// Context("Create an application"), func() {
	// 	It("returns 200 when the item is created", func() {
	// 		client := Mount(ApplicationsContext{Store: data}, (*ApplicationsContext).Save)
	// 		client.Post("/", App{

	// 		})
	// 		Expect(response.Code).To(Equal(200))
	// 		Expect(response.Code).To(Equal(200))
	// 	})
	// })
})
