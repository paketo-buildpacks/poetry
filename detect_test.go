package poetry_test

import (
	"os"
	"testing"

	packit "github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/poetry"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		detect packit.DetectFunc
	)

	it.Before(func() {
		detect = poetry.Detect()
	})

	it("returns a plan that provides poetry", func() {
		result, err := detect(packit.DetectContext{
			WorkingDir: "/working-dir",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: "poetry"},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: poetry.Pip,
						Metadata: poetry.BuildPlanMetadata{
							Build: true,
						},
					},
					{
						Name: poetry.CPython,
						Metadata: poetry.BuildPlanMetadata{
							Build: true,
						},
					},
				},
			},
		}))
	})

	context("when the BP_POETRY_VERSION is set", func() {
		it.Before(func() {
			Expect(os.Setenv("BP_POETRY_VERSION", "some-version")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("BP_POETRY_VERSION")).To(Succeed())
		})

		it("returns a plan that requires that version of poetry", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: "/working-dir",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(packit.DetectResult{
				Plan: packit.BuildPlan{
					Provides: []packit.BuildPlanProvision{
						{Name: "poetry"},
					},
					Requires: []packit.BuildPlanRequirement{
						{
							Name: poetry.Pip,
							Metadata: poetry.BuildPlanMetadata{
								Build: true,
							},
						},
						{
							Name: poetry.CPython,
							Metadata: poetry.BuildPlanMetadata{
								Build: true,
							},
						},
						{
							Name: "poetry",
							Metadata: poetry.BuildPlanMetadata{
								VersionSource: "BP_POETRY_VERSION",
								Version:       "some-version",
							},
						},
					},
				},
			}))
		})
	}, spec.Sequential())
}
