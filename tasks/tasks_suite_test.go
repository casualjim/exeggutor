package tasks_test

import (
	"fmt"
	"testing"

	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	//. "github.com/reverb/exeggutor/scheduler"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestState(t *testing.T) {
	RegisterFailHandler(Fail)
	pth := fmt.Sprintf("../test-reports/junit_exeggutor_tasks_%d.xml", config.GinkgoConfig.ParallelNode)
	junitReporter := reporters.NewJUnitReporter(pth)
	RunSpecsWithDefaultAndCustomReporters(t, "Exeggutor Tasks Test Suite", []Reporter{junitReporter})
}
