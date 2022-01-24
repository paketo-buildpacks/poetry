package poetry

import (
	"os"

	packit "github.com/paketo-buildpacks/packit/v2"
)

type BuildPlanMetadata struct {
	VersionSource string `toml:"version-source"`
	Build         bool   `toml:"build"`
	Version       string `toml:"version"`
}

func Detect() packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		requirements := []packit.BuildPlanRequirement{
			{
				Name: Pip,
				Metadata: BuildPlanMetadata{
					Build: true,
				},
			},
			{
				Name: CPython,
				Metadata: BuildPlanMetadata{
					Build: true,
				},
			},
		}

		if version, ok := os.LookupEnv("BP_POETRY_VERSION"); ok {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: PoetryDependency,
				Metadata: BuildPlanMetadata{
					VersionSource: "BP_POETRY_VERSION",
					Version:       version,
				},
			})
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: PoetryDependency},
				},
				Requires: requirements,
			},
		}, nil
	}
}
