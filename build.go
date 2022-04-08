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
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
//go:generate faux --interface DependencyManager --output fakes/dependency_manager.go
//go:generate faux --interface InstallProcess --output fakes/install_process.go
//go:generate faux --interface SitePackageProcess --output fakes/site_package_process.go

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

func Build(entryResolver EntryResolver, dependencyManager DependencyManager, installProcess InstallProcess, siteProcess SitePackageProcess, clock chronos.Clock, logs scribe.Emitter) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logs.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		logs.Process("Resolving Poetry version")
		entry, entries := entryResolver.Resolve(PoetryDependency, context.Plan.Entries, Priorities)
		logs.Candidates(entries)

		version, ok := entry.Metadata["version"].(string)
		if !ok {
			version = "default"
		}

		dependency, err := dependencyManager.Resolve(filepath.Join(context.CNBPath, "buildpack.toml"), entry.Name, version, context.Stack)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logs.SelectedDependency(entry, dependency, clock.Now())
		bom := dependencyManager.GenerateBillOfMaterials(dependency)

		poetryLayer, err := context.Layers.Get(PoetryLayerName)
		if err != nil {
			return packit.BuildResult{}, err
		}

		launch, build := entryResolver.MergeLayerTypes(PoetryDependency, context.Plan.Entries)

		var buildMetadata = packit.BuildMetadata{}
		var launchMetadata = packit.LaunchMetadata{}
		if build {
			buildMetadata = packit.BuildMetadata{BOM: bom}
		}

		if launch {
			launchMetadata = packit.LaunchMetadata{BOM: bom}
		}

		cachedSHA, ok := poetryLayer.Metadata[DependencySHAKey].(string)
		if ok && cachedSHA == dependency.SHA256 {
			logs.Process("Reusing cached layer %s", poetryLayer.Path)
			logs.Break()

			poetryLayer.Launch, poetryLayer.Build, poetryLayer.Cache = launch, build, build

			return packit.BuildResult{
				Layers: []packit.Layer{poetryLayer},
				Build:  buildMetadata,
				Launch: launchMetadata,
			}, nil
		}

		logs.Process("Executing build process")

		poetryLayer, err = poetryLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		poetryLayer.Launch, poetryLayer.Build, poetryLayer.Cache = launch, build, build

		logs.Subprocess("Installing Poetry %s", dependency.Version)
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
				panic(err)
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

		logs.Action("Completed in %s", duration.Round(time.Millisecond))
		logs.Break()

		logs.Process("Configuring environment")
		logs.Subprocess("%s", scribe.NewFormattedMapFromEnvironment(poetryLayer.SharedEnv))
		logs.Break()

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
