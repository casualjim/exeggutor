package exeggutor_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestState(t *testing.T) {
	RegisterFailHandler(Fail)
	pth := fmt.Sprintf("./test-reports/junit_%d.xml", config.GinkgoConfig.ParallelNode)
	junitReporter := reporters.NewJUnitReporter(pth)
	RunSpecsWithDefaultAndCustomReporters(t, "Exeggutor state Test Suite", []Reporter{junitReporter})
}
