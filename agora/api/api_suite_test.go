package api_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gocraft/web"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

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

func Get(route string, context interface{}, handler interface{}) {
	m := web.New(context)
	m.Get(route, handler)
	request, _ := http.NewRequest("GET", route, nil)
	response = httptest.NewRecorder()
	m.ServeHTTP(response, request)
}
