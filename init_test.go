package poetry_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnit(t *testing.T) {
	suite := spec.New("poetry", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Build", testBuild)
	suite("Detect", testDetect)
	suite("InstallProcess", testPoetryInstallProcess)
	suite("SiteProcess", testSiteProcess)
	suite.Run(t)
}
