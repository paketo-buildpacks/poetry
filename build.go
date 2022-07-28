package poetry

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface InstallProcess --output fakes/install_process.go
//go:generate faux --interface SitePackageProcess --output fakes/site_package_process.go
//go:generate faux --interface SBOMGenerator --output fakes/sbom_generator.go

// InstallProcess defines the interface for installing the poetry dependency into a layer.
type InstallProcess interface {
	Execute(version, targetLayerPath string) error
}

// SitePackageProcess defines the interface for looking site packages within a layer.
type SitePackageProcess interface {
	Execute(targetLayerPath string) (string, error)
}

type SBOMGenerator interface {
	Generate(dir string) (sbom.SBOM, error)
}

func Build(
	installProcess InstallProcess,
	siteProcess SitePackageProcess,
	sbomGenerator SBOMGenerator,
	clock chronos.Clock,
	logger scribe.Emitter,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		logger.Process("Resolving Poetry version")

		config, err := cargo.NewBuildpackParser().Parse(filepath.Join(context.CNBPath, "buildpack.toml"))
		if err != nil {
			return packit.BuildResult{}, err
		}

		entries := context.Plan.Entries
		entries = append(entries, packit.BuildpackPlanEntry{
			Name: PoetryDependency,
			Metadata: map[string]interface{}{
				"version":        config.Metadata.DefaultVersions[PoetryDependency],
				"version-source": DefaultVersions,
			},
		})

		planner := draft.NewPlanner()
		entry, entries := planner.Resolve(PoetryDependency, entries, Priorities)
		logger.Candidates(entries)

		version := entry.Metadata["version"].(string)
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

		cachedPoetryVersion, ok := poetryLayer.Metadata[PoetryVersion].(string)
		if ok && cachedPoetryVersion == version {
			logger.Process("Reusing cached layer %s", poetryLayer.Path)
			logger.Break()

			poetryLayer.Launch, poetryLayer.Build, poetryLayer.Cache = launch, build, build

			return packit.BuildResult{
				Layers: []packit.Layer{poetryLayer},
			}, nil
		}

		poetryLayer, err = poetryLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		poetryLayer.Launch, poetryLayer.Build, poetryLayer.Cache = launch, build, build

		logger.Process("Executing build process")
		logger.Subprocess("Installing Poetry %s", version)
		duration, err := clock.Measure(func() error {
			err = installProcess.Execute(version, poetryLayer.Path)
			if err != nil {
				return err
			}

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

		poetryLayer.Metadata = map[string]interface{}{
			PoetryVersion: version,
		}

		return packit.BuildResult{
			Layers: []packit.Layer{poetryLayer},
		}, nil
	}
}
