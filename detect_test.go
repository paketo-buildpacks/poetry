package poetry_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/poetry"
	"github.com/paketo-buildpacks/poetry/fakes"
	"github.com/sclevine/spec"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		parsePythonVersion *fakes.PyProjectPythonVersionParser

		workingDir string

		detect packit.DetectFunc
	)

	it.Before(func() {
		parsePythonVersion = &fakes.PyProjectPythonVersionParser{}

		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		detect = poetry.Detect(parsePythonVersion)
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("when pyproject.toml is present", func() {
		it.Before(func() {
			Expect(os.WriteFile(filepath.Join(workingDir, "pyproject.toml"), []byte(""), 0755)).To(Succeed())
		})

		it("returns a plan that provides poetry", func() {
			parsePythonVersion.ParsePythonVersionCall.Returns.String = "1.2.3"

			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
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
					WorkingDir: workingDir,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(parsePythonVersion.ParsePythonVersionCall.Receives.String).To(Equal(filepath.Join(workingDir, "pyproject.toml")))
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
		})

		context("error handling", func() {
			// the python dependency is mandatory
			// https://python-poetry.org/docs/pyproject/#dependencies-and-dev-dependencies
			it("fails detection when no python version found", func() {
				parsePythonVersion.ParsePythonVersionCall.Returns.String = ""

				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(packit.Fail.WithMessage("pyproject.toml must include [tool.poetry.dependencies.python], see https://python-poetry.org/docs/pyproject/#dependencies-and-dev-dependencies")))
			})

			it("handles an error from the pyproject.toml parser", func() {
				expectedErr := errors.New("hi")
				parsePythonVersion.ParsePythonVersionCall.Returns.Error = expectedErr

				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(Equal(expectedErr))
			})
		})
	})

	context("when pyproject.toml is not present", func() {
		it("fails detection", func() {
			_, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).To(MatchError("pyproject.toml is not present"))
		})
	})
}
