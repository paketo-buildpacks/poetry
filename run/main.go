package main

import (
	"os"

	packit "github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/poetry"
)

func main() {
	entryResolver := draft.NewPlanner()
	dependencyManager := postal.NewService(cargo.NewTransport())
	installProcess := poetry.NewPoetryInstallProcess(pexec.NewExecutable("pip"))
	siteProcess := poetry.NewSiteProcess(pexec.NewExecutable("python"))
	logs := scribe.NewEmitter(os.Stdout)
	pyProjectParser := poetry.NewPyProjectParser()

	packit.Run(
		poetry.Detect(pyProjectParser),
		poetry.Build(entryResolver, dependencyManager, installProcess, siteProcess, chronos.DefaultClock, logs),
	)
}
