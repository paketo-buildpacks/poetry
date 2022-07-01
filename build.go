package poetry

import (
	"fmt"
	"strings"
	"time"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface InstallProcess --output fakes/install_process.go
//go:generate faux --interface VersionProcess --output fakes/version_process.go
//go:generate faux --interface SitePackageProcess --output fakes/site_package_process.go
//go:generate faux --interface SBOMGenerator --output fakes/sbom_generator.go

// InstallProcess defines the interface for installing the poetry dependency into a layer.
type InstallProcess interface {
	Execute(version, targetLayerPath string) error
}

// VersionProcess defines the interface for retrieving the poetry version
type VersionProcess interface {
	Execute(targetLayerPath string) (string, error)
}

// SitePackageProcess defines the interface for looking site packages within a layer.
type SitePackageProcess interface {
	Execute(targetLayerPath string) (string, error)
}

//go:generate faux --interface SBOMGenerator --output fakes/sbom_generator.go
type SBOMGenerator interface {
	Generate(dir string) (sbom.SBOM, error)
}

func Build(
	installProcess InstallProcess,
	versionProcess VersionProcess,
	siteProcess SitePackageProcess,
	sbomGenerator SBOMGenerator,
	clock chronos.Clock,
	logger scribe.Emitter,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		logger.Process("Resolving Poetry version")
		planner := draft.NewPlanner()
		entry, entries := planner.Resolve(PoetryDependency, context.Plan.Entries, Priorities)
		logger.Candidates(entries)

		version, ok := entry.Metadata["version"].(string)
		if !ok {
			version = ""
		}

		source, ok := entry.Metadata["version-source"].(string)
		if !ok {
			source = "<unknown>"
		}

		logger.Subprocess("Selected Poetry version (using %s): %s", source, version)

		poetryLayer, err := context.Layers.Get(PoetryLayerName)
		if err != nil {
			return packit.BuildResult{}, err
		}

		launch, build := planner.MergeLayerTypes(PoetryDependency, context.Plan.Entries)

		poetryLayer, err = poetryLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		poetryLayer.Launch, poetryLayer.Build, poetryLayer.Cache = launch, build, build

		logger.Process("Executing build process")
		duration, err := clock.Measure(func() error {
			err = installProcess.Execute(version, poetryLayer.Path)
			if err != nil {
				return err
			}

			var installedVersion string
			installedVersion, err = versionProcess.Execute(poetryLayer.Path)
			if err != nil {
				return err
			}

			logger.Subprocess("Installed Poetry %s", installedVersion)

			// Look up the site packages path and prepend it onto $PYTHONPATH
			sitePackagesPath, err := siteProcess.Execute(poetryLayer.Path)
			if err != nil {
				return fmt.Errorf("failed to locate site packages in poetry layer: %w", err)
			}

			if sitePackagesPath == "" {
				return fmt.Errorf("poetry installation failed: site packages are missing from the poetry layer")
			}

			poetryLayer.SharedEnv.Prepend("PYTHONPATH", strings.TrimRight(sitePackagesPath, "\n"), ":")

			return nil
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		logger.GeneratingSBOM(poetryLayer.Path)
		var sbomContent sbom.SBOM
		duration, err = clock.Measure(func() error {
			sbomContent, err = sbomGenerator.Generate(poetryLayer.Path)
			return err
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		logger.FormattingSBOM(context.BuildpackInfo.SBOMFormats...)
		poetryLayer.SBOM, err = sbomContent.InFormats(context.BuildpackInfo.SBOMFormats...)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.EnvironmentVariables(poetryLayer)

		return packit.BuildResult{
			Layers: []packit.Layer{poetryLayer},
		}, nil
	}
}
