package api_test

import (
	"encoding/json"
	"fmt"
	stdlog "log"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/op/go-logging"
	. "github.com/reverb/exeggutor/agora/api"
	"github.com/reverb/exeggutor/store"
)

func testApp(name, component string, context *APIContext) App {
	app := App{
		Name: name,
		Components: map[string]*AppComponent{component: &AppComponent{
			Name:          component,
			Cpus:          1,
			Mem:           1,
			DistURL:       "http://somewhere.com",
			Command:       "./" + component,
			Ports:         map[string]int{"HTTP": 8000},
			Version:       "0.0.1",
			ComponentType: "service",
		}},
	}
	return app
}

var _ = Describe("ApplicationsApi", func() {

	var (
		context    *APIContext
		controller *ApplicationsController
		server     *testHTTP
	)

	BeforeEach(func() {
		logBackend := logging.NewLogBackend(os.Stderr, "", stdlog.LstdFlags|stdlog.Lshortfile)
		logBackend.Color = true
		logging.SetBackend(logBackend)
		context = &APIContext{
			Config:   testAppConfig(),
			AppStore: store.NewEmptyInMemoryStore(),
		}
		context.AppStore.Start()
		controller = NewApplicationsController(context)
		server = NewTestHTTP()
		server.Mount("GET", "/applications", controller.ListAll)
		server.Mount("GET", "/applications/:name", controller.ShowOne)
		server.Mount("POST", "/applications", controller.Save)
		server.Mount("PUT", "/applications/:name", controller.Save)
		server.Mount("DELETE", "/applications/:name", controller.Delete)
	})

	AfterEach(func() {
		context.AppStore.Stop()
	})

	Context("List all applications", func() {
		It("returns a 200 Status Code", func() {
			server.Get("/applications")
			Expect(response.Code).To(Equal(200))
			Expect(response.Body.String()).To(Equal("[]"))
		})

		It("returns a list of applications", func() {
			expected := []App{
				testApp("bifrost-service", "bifrost-service", context),
				testApp("veggr-service", "veggr-service", context),
			}
			for _, app := range expected {
				bytes, err := json.Marshal(&app)
				if err != nil {
					Fail(err.Error())
				}
				context.AppStore.Set(app.Name, bytes)
			}
			server.Get("/applications")
			Expect(response.Code).To(Equal(200))
			bodyBytes := response.Body.Bytes()

			var apps []App
			err := json.Unmarshal(bodyBytes, &apps)
			Expect(err).ToNot(HaveOccurred())
			Expect(apps).To(HaveLen(2))
			Expect(apps).To(Equal(expected))
		})
	})

	Context("Get a single application", func() {

		It("returns 200 and the application", func() {
			expected := testApp("foo393", "foo939", context)
			expectedJson, _ := json.Marshal(expected)
			controller.AppStore = store.NewInMemoryStore(map[string][]byte{expected.Name: expectedJson})

			server.Get("/applications/" + expected.Name)

			Expect(response.Code).To(Equal(200))

			var app App
			err := json.Unmarshal(response.Body.Bytes(), &app)
			Expect(err).NotTo(HaveOccurred())
			Expect(app).To(Equal(expected))
		})

		It("returns 404 and an error message", func() {
			expected := testApp("foo393", "foo939", context)
			server.Get("/applications/" + expected.Name)

			Expect(response.Code).To(Equal(404))
			Expect(response.Body.Bytes()).To(MatchJSON(fmt.Sprintf(`{"message":"Couldn't find %s for key '%s'."}`, "App", expected.Name)))
		})

	})

	Context("Create an application", func() {
		It("returns 200 when the item is created", func() {
			expected := testApp("blah-service", "blah", context)
			server.Post("/applications", expected)
			Expect(response.Code).To(Equal(200))
			bodyBytes := response.Body.Bytes()
			var actual App
			err := json.Unmarshal(bodyBytes, &actual)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(expected))
		})

		It("returns 422 when the app is invalid ", func() {
			expected := App{}
			server.Post("/applications", expected)
			Expect(response.Code).To(Equal(422))
		})
	})

	Context("Update an application", func() {
		It("returns 200 when the item is updated", func() {
			expected := testApp("blah-service", "blah", context)
			expectedJson, _ := json.Marshal(expected)
			controller.AppStore = store.NewInMemoryStore(map[string][]byte{expected.Name: expectedJson})

			server.Put("/applications/"+expected.Name, expected)
			Expect(response.Code).To(Equal(200))

			bodyBytes := response.Body.Bytes()
			var actual App
			err := json.Unmarshal(bodyBytes, &actual)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(expected))
		})

		It("returns 422 when the app is invalid ", func() {
			expected := App{Name: "blah-service"}
			expectedJson, _ := json.Marshal(expected)
			controller.AppStore = store.NewInMemoryStore(map[string][]byte{expected.Name: expectedJson})
			server.Put("/applications/"+expected.Name, expected)
			Expect(response.Code).To(Equal(422))
		})

	})

	Context("Delete an application", func() {
		It("returns 204 when the delete succeeds", func() {
			expected := testApp("blah-service", "blah", context)
			expectedJson, _ := json.Marshal(expected)
			controller.AppStore = store.NewInMemoryStore(map[string][]byte{expected.Name: expectedJson})

			server.Delete("/applications/" + expected.Name)
			Expect(response.Code).To(Equal(204))

			server.Get("/application/" + expected.Name)
			Expect(response.Code).To(Equal(404))
		})

		It("returns 204 when the doesn't exist", func() {
			expected := testApp("blah-service", "blah", context)

			server.Delete("/applications/" + expected.Name)
			Expect(response.Code).To(Equal(204))

			server.Get("/application/" + expected.Name)
			Expect(response.Code).To(Equal(404))
		})
	})

})
