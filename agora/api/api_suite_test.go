package api

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/julienschmidt/httprouter"
	"github.com/reverb/exeggutor"
)

var (
	response *httptest.ResponseRecorder
)

func testAppConfig() *exeggutor.Config {
	data, _ := ioutil.TempDir("", "agora-api-data")
	logs, _ := ioutil.TempDir("", "agora-api-logs")
	work, _ := ioutil.TempDir("", "agora-api-work")
	return &exeggutor.Config{
		ZookeeperURL:    "zk://localhost:2181/apiTests",
		MesosMaster:     "zk://localhost:2181/mesos",
		DataDirectory:   data,
		LogDirectory:    logs,
		StaticFiles:     "./static/build",
		WorkDirectory:   work,
		ConfigDirectory: "./etc",
		Port:            9484,
		Interface:       "0.0.0.0",
		Mode:            "test",
		DockerIndex: &exeggutor.DockerIndexConfig{
			Host:       "dev-docker.helloreverb.com",
			Port:       443,
			Scheme:     "https",
			APIVersion: "v1",
		},
	}
}

type testHTTP struct {
	router *httprouter.Router
}

func NewTestHTTP() *testHTTP {
	return &testHTTP{router: httprouter.New()}
}

func (t *testHTTP) Mount(method, pattern string, handler httprouter.Handle) {
	t.router.Handle(method, pattern, handler)
}

func (t *testHTTP) Get(route string) {
	request, _ := http.NewRequest("GET", route, nil)
	request.Header.Set("Content-Type", JSONContentType)
	response = httptest.NewRecorder()
	t.router.ServeHTTP(response, request)
}

func (t *testHTTP) Post(route string, data interface{}) {
	var d []byte
	if data != nil {
		d, _ = json.Marshal(data)
	}
	request, _ := http.NewRequest("POST", route, bytes.NewBuffer(d))
	request.Header.Set("Content-Type", JSONContentType)

	response = httptest.NewRecorder()
	t.router.ServeHTTP(response, request)
}

func (t *testHTTP) Put(route string, data interface{}) {
	var d []byte
	if data != nil {
		d, _ = json.Marshal(data)
	}
	request, _ := http.NewRequest("PUT", route, bytes.NewBuffer(d))
	request.Header.Set("Content-Type", JSONContentType)

	response = httptest.NewRecorder()
	t.router.ServeHTTP(response, request)
}

func (t *testHTTP) Delete(route string) {
	request, _ := http.NewRequest("DELETE", route, nil)
	request.Header.Set("Content-Type", JSONContentType)

	response = httptest.NewRecorder()
	t.router.ServeHTTP(response, request)
}
