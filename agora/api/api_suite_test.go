package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gocraft/web"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/reverb/exeggutor/agora/middlewares"

	"testing"
)

var (
	response *httptest.ResponseRecorder
)

func TestState(t *testing.T) {
	RegisterFailHandler(Fail)
	pth := fmt.Sprintf("../../test-reports/junit_executor_agora_api_%d.xml", config.GinkgoConfig.ParallelNode)
	junitReporter := reporters.NewJUnitReporter(pth)
	RunSpecsWithDefaultAndCustomReporters(t, "Exeggutor state Test Suite", []Reporter{junitReporter})
}

type TestInterface struct {
	Context interface{}
	Handler interface{}
}

func Mount(context, handler interface{}) TestInterface {
	return TestInterface{Context: context, Handler: handler}
}

func (t TestInterface) Get(route string) {
	m := web.New(t.Context)
	m.Get(route, t.Handler)
	request, _ := http.NewRequest("GET", route, nil)
	request.Header.Set("Content-Type", middlewares.JSONContentType)
	response = httptest.NewRecorder()
	m.ServeHTTP(response, request)
}

func (t TestInterface) Post(route string, data interface{}) {
	m := web.New(t.Context)
	m.Get(route, t.Handler)
	var d []byte
	if data != nil {
		d, _ = json.Marshal(data)
	}
	request, _ := http.NewRequest("POST", route, bytes.NewBuffer(d))
	request.Header.Set("Content-Type", middlewares.JSONContentType)

	response = httptest.NewRecorder()
	m.ServeHTTP(response, request)
}
