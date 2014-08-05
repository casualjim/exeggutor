package api

import (
	"encoding/json"
	"fmt"
	stdlog "log"
	"os"
	"testing"

	"github.com/op/go-logging"
	"github.com/reverb/exeggutor/agora/api/model"
	"github.com/reverb/exeggutor/store"
	app_store "github.com/reverb/exeggutor/store/apps"
	. "github.com/smartystreets/goconvey/convey"
)

func testApp(name, component string, context *APIContext) model.App {
	app := model.App{
		Name: name,
		Components: map[string]model.AppComponent{component: model.AppComponent{
			Name:          component,
			Cpus:          1,
			Mem:           1,
			DistURL:       fmt.Sprintf("docker://dev-docker.helloreverb.com/v1/%s/%s:0.0.1", name, component),
			Command:       "./" + component,
			Ports:         map[string]int{"HTTP": 8000},
			Env:           make(map[string]string),
			Version:       "0.0.1",
			ComponentType: "service",
		}},
	}
	return app
}

func TestApplicationsApi(t *testing.T) {

	Convey("ApplicationsApi", t, func() {
		logBackend := logging.NewLogBackend(os.Stderr, "", stdlog.LstdFlags|stdlog.Lshortfile)
		logBackend.Color = true
		logging.SetBackend(logBackend)
		logging.SetLevel(logging.ERROR, "")
		context := &APIContext{
			Config:   testAppConfig(),
			AppStore: app_store.NewWithStore(store.NewEmptyInMemoryStore()),
		}
		context.AppStore.Start()
		controller := NewApplicationsController(context)
		converter := model.New(context.Config)
		server := NewTestHTTP()
		server.Mount("GET", "/applications", controller.ListAll)
		server.Mount("GET", "/applications/:name", controller.ShowOne)
		server.Mount("POST", "/applications", controller.Save)
		server.Mount("PUT", "/applications/:name", controller.Save)
		server.Mount("DELETE", "/applications/:name", controller.Delete)

		Reset(func() {
			context.AppStore.Stop()
		})

		Convey("List all applications", func() {
			Convey("returns a 200 Status Code", func() {
				server.Get("/applications")
				So(response.Code, ShouldEqual, 200)
				So(response.Body.String(), ShouldEqual, "[]")
			})

			Convey("returns a list of applications", func() {
				expected := []model.App{
					testApp("bifrost-service", "bifrost-service", context),
					testApp("veggr-service", "veggr-service", context),
				}
				for _, app := range expected {
					for _, a := range converter.ToAppManifest(&app) {
						context.AppStore.Save(&a)
					}
				}
				server.Get("/applications")
				So(response.Code, ShouldEqual, 200)
				bodyBytes := response.Body.Bytes()

				var apps []model.App
				err := json.Unmarshal(bodyBytes, &apps)
				So(err, ShouldBeNil)
				So(len(apps), ShouldEqual, 2)

				So(apps, ShouldResemble, expected)
			})
		})

		Convey("Get a single application", func() {

			Convey("returns 200 and the application", func() {
				ex := testApp("foo393", "foo939", context)
				expected := converter.ToAppManifest(&ex)[0]
				context.AppStore.Save(&expected)

				server.Get("/applications/" + expected.GetId())

				So(response.Code, ShouldEqual, 200)

				var app model.App
				err := json.Unmarshal(response.Body.Bytes(), &app)
				So(err, ShouldBeNil)
				So(app, ShouldResemble, ex)
			})

			Convey("returns 404 and an error message", func() {
				ex := testApp("foo393", "foo939", context)
				expected := converter.ToAppManifest(&ex)[0]
				server.Get("/applications/" + expected.GetId())

				So(response.Code, ShouldEqual, 404)
				So(response.Body.String(), ShouldEqual, fmt.Sprintf(`{"message":"Couldn't find %s for key '%s'.", "type": "error"}`, "App", expected.GetId()))
			})

		})

		Convey("Create an application", func() {
			Convey("returns 200 when the item is created", func() {
				expected := testApp("blah-service", "blah", context)

				server.Post("/applications", expected)
				So(response.Code, ShouldEqual, 200)
				bodyBytes := response.Body.Bytes()
				var actual model.App
				err := json.Unmarshal(bodyBytes, &actual)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, expected)
			})

			Convey("returns 422 when the app is invalid ", func() {
				expected := model.App{}
				server.Post("/applications", expected)
				So(response.Code, ShouldEqual, 422)
			})
		})

		Convey("Update an application", func() {
			Convey("returns 200 when the item is updated", func() {
				expected := testApp("blah-service", "blah", context)
				expectedJSON, _ := json.Marshal(expected)
				controller.AppStore = app_store.NewWithStore(store.NewInMemoryStore(map[string][]byte{expected.Name: expectedJSON}))

				server.Put("/applications/"+expected.Name, expected)
				So(response.Code, ShouldEqual, 200)

				bodyBytes := response.Body.Bytes()
				var actual model.App
				err := json.Unmarshal(bodyBytes, &actual)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, expected)
			})

			Convey("returns 422 when the app is invalid ", func() {
				expected := model.App{Name: "blah-service"}
				expectedJSON, _ := json.Marshal(expected)
				controller.AppStore = app_store.NewWithStore(store.NewInMemoryStore(map[string][]byte{expected.Name: expectedJSON}))
				server.Put("/applications/"+expected.Name, expected)
				So(response.Code, ShouldEqual, 422)
			})

		})

		Convey("Delete an application", func() {
			Convey("returns 204 when the delete succeeds", func() {
				expected := testApp("blah-service", "blah", context)
				expectedJSON, _ := json.Marshal(expected)
				controller.AppStore = app_store.NewWithStore(store.NewInMemoryStore(map[string][]byte{expected.Name: expectedJSON}))

				server.Delete("/applications/" + expected.Name)
				So(response.Code, ShouldEqual, 204)

				server.Get("/application/" + expected.Name)
				So(response.Code, ShouldEqual, 404)
			})

			Convey("returns 204 when the doesn't exist", func() {
				expected := testApp("blah-service", "blah", context)

				server.Delete("/applications/" + expected.Name)
				So(response.Code, ShouldEqual, 204)

				server.Get("/application/" + expected.Name)
				So(response.Code, ShouldEqual, 404)
			})
		})

	})
}
