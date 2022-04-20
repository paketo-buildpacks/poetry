package poetry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	packit "github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
//go:generate faux --interface DependencyManager --output fakes/dependency_manager.go
//go:generate faux --interface InstallProcess --output fakes/install_process.go
//go:generate faux --interface SitePackageProcess --output fakes/site_package_process.go
//go:generate faux --interface SBOMGenerator --output fakes/sbom_generator.go

type EntryResolver interface {
	Resolve(name string, entries []packit.BuildpackPlanEntry, priorites []interface{}) (packit.BuildpackPlanEntry, []packit.BuildpackPlanEntry)
	MergeLayerTypes(name string, entries []packit.BuildpackPlanEntry) (launch, build bool)
}

type DependencyManager interface {
	Resolve(path, id, version, stack string) (postal.Dependency, error)
	Deliver(dependency postal.Dependency, cnbPath, destinationPath, platformPath string) error
	GenerateBillOfMaterials(dependencies ...postal.Dependency) []packit.BOMEntry
}

// InstallProcess defines the interface for installing the poetry dependency into a layer.
type InstallProcess interface {
	Execute(srcPath, targetLayerPath string) error
}

// SitePackageProcess defines the interface for looking site packages within a layer.
type SitePackageProcess interface {
	Execute(targetLayerPath string) (string, error)
}

type SBOMGenerator interface {
	GenerateFromDependency(dependency postal.Dependency, dir string) (sbom.SBOM, error)
}

func Build(
	entryResolver EntryResolver,
	dependencyManager DependencyManager,
	installProcess InstallProcess,
	siteProcess SitePackageProcess,
	sbomGenerator SBOMGenerator,
	clock chronos.Clock,
	logger scribe.Emitter,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		logger.Process("Resolving Poetry version")
		entry, entries := entryResolver.Resolve(PoetryDependency, context.Plan.Entries, Priorities)
		logger.Candidates(entries)

		version, ok := entry.Metadata["version"].(string)
		if !ok {
			version = "default"
		}

		dependency, err := dependencyManager.Resolve(filepath.Join(context.CNBPath, "buildpack.toml"), entry.Name, version, context.Stack)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.SelectedDependency(entry, dependency, clock.Now())
		legacySBOM := dependencyManager.GenerateBillOfMaterials(dependency)

		poetryLayer, err := context.Layers.Get(PoetryLayerName)
		if err != nil {
			return packit.BuildResult{}, err
		}

		launch, build := entryResolver.MergeLayerTypes(PoetryDependency, context.Plan.Entries)

		var buildMetadata = packit.BuildMetadata{}
		var launchMetadata = packit.LaunchMetadata{}
		if build {
			buildMetadata = packit.BuildMetadata{BOM: legacySBOM}
		}

		if launch {
			launchMetadata = packit.LaunchMetadata{BOM: legacySBOM}
		}

		cachedSHA, ok := poetryLayer.Metadata[DependencySHAKey].(string)
		if ok && cachedSHA == dependency.SHA256 {
			logger.Process("Reusing cached layer %s", poetryLayer.Path)
			logger.Break()

			poetryLayer.Launch, poetryLayer.Build, poetryLayer.Cache = launch, build, build

			return packit.BuildResult{
				Layers: []packit.Layer{poetryLayer},
				Build:  buildMetadata,
				Launch: launchMetadata,
			}, nil
		}

		logger.Process("Executing build process")

		poetryLayer, err = poetryLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		poetryLayer.Launch, poetryLayer.Build, poetryLayer.Cache = launch, build, build

		logger.Subprocess("Installing Poetry %s", dependency.Version)
		duration, err := clock.Measure(func() error {
			// Install the poetry source to a temporary dir, since we only need access to
			// it as an intermediate step when installing poetry.
			// It doesn't need to go into a layer, since we won't need it in future builds.
			poetrySrcDir, err := os.MkdirTemp("", "poetry-source")
			if err != nil {
				return fmt.Errorf("failed to create temp poetry-source dir: %w", err)
			}

			err = dependencyManager.Deliver(dependency, context.CNBPath, poetrySrcDir, context.Platform.Path)
			if err != nil {
				return err
			}

			err = installProcess.Execute(poetrySrcDir, poetryLayer.Path)
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
			sbomContent, err = sbomGenerator.GenerateFromDependency(dependency, poetryLayer.Path)
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

		logger.Process("Configuring environment")
		logger.Subprocess("%s", scribe.NewFormattedMapFromEnvironment(poetryLayer.SharedEnv))
		logger.Break()

		poetryLayer.Metadata = map[string]interface{}{
			DependencySHAKey: dependency.SHA256,
			"built_at":       clock.Now().Format(time.RFC3339Nano),
		}

		return packit.BuildResult{
			Layers: []packit.Layer{poetryLayer},
			Build:  buildMetadata,
			Launch: launchMetadata,
		}, nil
	}
}
