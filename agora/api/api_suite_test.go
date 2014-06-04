package api_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestState(t *testing.T) {
	RegisterFailHandler(Fail)
	pth := fmt.Sprintf("../test-reports/junit_executor_agora_api_%d.xml", config.GinkgoConfig.ParallelNode)
	junitReporter := reporters.NewJUnitReporter(pth)
	RunSpecsWithDefaultAndCustomReporters(t, "Exeggutor state Test Suite", []Reporter{junitReporter})
}

func Request(method string, route string, handler martini.Handler) {
	// m := martini.Classic()
	// m.Get(route, handler)
	// m.Use(render.Renderer())
	request, _ := http.NewRequest(method, route, nil)
	response = httptest.NewRecorder()
	m.ServeHTTP(response, request)
}

func PostRequest(method string, route string, handler martini.Handler, body io.Reader) {
	m := martini.Classic()
	m.Post(route, binding.Json(Todo{}), handler)
	m.Use(render.Renderer())
	request, _ := http.NewRequest(method, route, body)
	response = httptest.NewRecorder()
	m.ServeHTTP(response, request)
}
