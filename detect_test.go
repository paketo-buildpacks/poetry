package poetry_test

import (
	"errors"
	"github.com/paketo-buildpacks/poetry/fakes"
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

		parsePythonVersion *fakes.PyProjectPythonVersionParser

		detect packit.DetectFunc
	)

	it.Before(func() {
		parsePythonVersion = &fakes.PyProjectPythonVersionParser{}

		detect = poetry.Detect(parsePythonVersion)
	})

	it("returns a plan that provides poetry", func() {
		parsePythonVersion.ParsePythonVersionCall.Returns.String = "1.2.3"

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
							Build:         true,
							Version:       "1.2.3",
							VersionSource: "pyproject.toml",
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
			parsePythonVersion.ParsePythonVersionCall.Returns.String = "9.8.7"

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
								Build:         true,
								Version:       "9.8.7",
								VersionSource: "pyproject.toml",
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

	context("handles pyproject.toml parser", func() {
		it("passes WorkingDir to the pyproject.toml parser #1", func() {
			_, _ = detect(packit.DetectContext{
				WorkingDir: "/hi",
			})
			Expect(parsePythonVersion.ParsePythonVersionCall.Receives.Path).To(Equal("/hi"))
		})

		it("passes WorkingDir to the pyproject.toml parser #2", func() {
			_, _ = detect(packit.DetectContext{
				WorkingDir: "/other",
			})
			Expect(parsePythonVersion.ParsePythonVersionCall.Receives.Path).To(Equal("/other"))
		})

		it("handles an error from the pyproject.toml parser #1", func() {
			expectedErr := errors.New("hi")
			parsePythonVersion.ParsePythonVersionCall.Returns.Error = expectedErr

			result, err := detect(packit.DetectContext{
				WorkingDir: "/working-dir",
			})
			Expect(result).To(Equal(packit.DetectResult{}))
			Expect(err).To(Equal(expectedErr))
		})

		it("handles an error from the pyproject.toml parser #2", func() {
			expectedErr := errors.New("other")
			parsePythonVersion.ParsePythonVersionCall.Returns.Error = expectedErr

			result, err := detect(packit.DetectContext{
				WorkingDir: "/working-dir",
			})
			Expect(result).To(Equal(packit.DetectResult{}))
			Expect(err).To(Equal(expectedErr))
		})
	})
}
