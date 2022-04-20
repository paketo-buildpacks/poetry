package main

import (
	"os"

	packit "github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/poetry"
)

type Generator struct{}

func (f Generator) GenerateFromDependency(dependency postal.Dependency, path string) (sbom.SBOM, error) {
	return sbom.GenerateFromDependency(dependency, path)
}

func main() {
	logger := scribe.NewEmitter(os.Stdout).WithLevel(os.Getenv("BP_LOG_LEVEL"))

	packit.Run(
		poetry.Detect(poetry.NewPyProjectParser()),
		poetry.Build(
			draft.NewPlanner(),
			postal.NewService(cargo.NewTransport()),
			poetry.NewPoetryInstallProcess(pexec.NewExecutable("pip")),
			poetry.NewSiteProcess(pexec.NewExecutable("python")),
			Generator{},
			chronos.DefaultClock,
			logger,
		),
	)
}
