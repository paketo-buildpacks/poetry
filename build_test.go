package poetry_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"

	//nolint Ignore SA1019, informed usage of deprecated package
	"github.com/paketo-buildpacks/packit/v2/paketosbom"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/poetry"
	"github.com/paketo-buildpacks/poetry/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir string
		cnbDir    string

		dependencyManager *fakes.DependencyManager
		installProcess    *fakes.InstallProcess
		siteProcess       *fakes.SitePackageProcess
		sbomGenerator     *fakes.SBOMGenerator

		buffer *bytes.Buffer

		build        packit.BuildFunc
		buildContext packit.BuildContext
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		// Legacy SBOM
		dependencyManager = &fakes.DependencyManager{}
		dependencyManager.ResolveCall.Returns.Dependency = postal.Dependency{
			ID:       "poetry",
			Name:     "poetry-dependency-name",
			Checksum: "poetry-dependency-sha",
			Stacks:   []string{"some-stack"},
			URI:      "poetry-dependency-uri",
			Version:  "poetry-dependency-version",
		}

		dependencyManager.GenerateBillOfMaterialsCall.Returns.BOMEntrySlice = []packit.BOMEntry{
			{
				Name: "poetry",
				Metadata: paketosbom.BOMMetadata{
					Version: "poetry-dependency-version",
					Checksum: paketosbom.BOMChecksum{
						Algorithm: paketosbom.SHA256,
						Hash:      "poetry-dependency-sha",
					},
					URI: "poetry-dependency-uri",
				},
			},
		}

		// Syft SBOM
		sbomGenerator = &fakes.SBOMGenerator{}
		sbomGenerator.GenerateFromDependencyCall.Returns.SBOM = sbom.SBOM{}

		installProcess = &fakes.InstallProcess{}
		siteProcess = &fakes.SitePackageProcess{}
		siteProcess.ExecuteCall.Returns.String = filepath.Join(layersDir, "poetry", "lib", "python3.8", "site-packages")

		buffer = bytes.NewBuffer(nil)
		logEmitter := scribe.NewEmitter(buffer)

		build = poetry.Build(
			dependencyManager,
			installProcess,
			siteProcess,
			sbomGenerator,
			chronos.DefaultClock,
			logEmitter,
		)

		buildContext = packit.BuildContext{
			BuildpackInfo: packit.BuildpackInfo{
				Name:        "Some Buildpack",
				Version:     "some-version",
				SBOMFormats: []string{sbom.CycloneDXFormat, sbom.SPDXFormat},
			},
			CNBPath: cnbDir,
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{Name: "poetry"},
				},
			},
			Platform: packit.Platform{Path: "platform"},
			Layers:   packit.Layers{Path: layersDir},
			Stack:    "some-stack",
		}
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
	})

	it("returns a result that installs poetry", func() {
		result, err := build(buildContext)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(1))
		layer := result.Layers[0]

		Expect(layer.Name).To(Equal("poetry"))
		Expect(layer.Path).To(Equal(filepath.Join(layersDir, "poetry")))

		Expect(layer.SharedEnv).To(HaveLen(2))
		Expect(layer.SharedEnv["PYTHONPATH.delim"]).To(Equal(":"))
		Expect(layer.SharedEnv["PYTHONPATH.prepend"]).To(Equal(filepath.Join(layersDir, "poetry", "lib/python3.8/site-packages")))

		Expect(layer.BuildEnv).To(BeEmpty())
		Expect(layer.LaunchEnv).To(BeEmpty())
		Expect(layer.ProcessLaunchEnv).To(BeEmpty())

		Expect(layer.Build).To(BeFalse())
		Expect(layer.Launch).To(BeFalse())
		Expect(layer.Cache).To(BeFalse())

		Expect(layer.Metadata).To(HaveLen(1))
		Expect(layer.Metadata["dependency-checksum"]).To(Equal("poetry-dependency-sha"))

		Expect(layer.SBOM.Formats()).To(HaveLen(2))
		var actualExtensions []string
		for _, format := range layer.SBOM.Formats() {
			actualExtensions = append(actualExtensions, format.Extension)
		}
		Expect(actualExtensions).To(ConsistOf("cdx.json", "spdx.json"))

		Expect(dependencyManager.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
		Expect(dependencyManager.ResolveCall.Receives.Id).To(Equal("poetry"))
		Expect(dependencyManager.ResolveCall.Receives.Version).To(Equal("default"))
		Expect(dependencyManager.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyManager.GenerateBillOfMaterialsCall.CallCount).To(Equal(1))
		Expect(dependencyManager.GenerateBillOfMaterialsCall.Receives.Dependencies).To(Equal([]postal.Dependency{
			{
				ID:       "poetry",
				Name:     "poetry-dependency-name",
				Checksum: "poetry-dependency-sha",
				Stacks:   []string{"some-stack"},
				URI:      "poetry-dependency-uri",
				Version:  "poetry-dependency-version",
			},
		}))

		Expect(sbomGenerator.GenerateFromDependencyCall.Receives.Dir).To(Equal(filepath.Join(layersDir, "poetry")))

		Expect(installProcess.ExecuteCall.Receives.Version).To(ContainSubstring("poetry-dependency-version"))
		Expect(installProcess.ExecuteCall.Receives.TargetLayerPath).To(Equal(filepath.Join(layersDir, "poetry")))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Resolving Poetry version"))
		Expect(buffer.String()).To(ContainSubstring("Selected poetry-dependency-name version (using <unknown>): poetry-dependency-version"))
		Expect(buffer.String()).To(ContainSubstring("Executing build process"))
		Expect(buffer.String()).To(ContainSubstring("Installing Poetry poetry-dependency-version"))
		Expect(buffer.String()).To(ContainSubstring("Completed in"))
	})

	context("when the plan entry requires the dependency during the build and launch phases", func() {
		it.Before(func() {
			buildContext.Plan.Entries[0].Metadata = map[string]interface{}{
				"launch": true,
				"build":  true,
			}
		})

		it("makes the layer available in those phases", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			layer := result.Layers[0]

			Expect(layer.Name).To(Equal("poetry"))

			Expect(layer.Build).To(BeTrue())
			Expect(layer.Launch).To(BeTrue())
			Expect(layer.Cache).To(BeTrue())

			Expect(result.Build.BOM).To(Equal(
				[]packit.BOMEntry{
					{
						Name: "poetry",
						Metadata: paketosbom.BOMMetadata{
							Checksum: paketosbom.BOMChecksum{
								Algorithm: paketosbom.SHA256,
								Hash:      "poetry-dependency-sha",
							},
							URI:     "poetry-dependency-uri",
							Version: "poetry-dependency-version",
						},
					},
				},
			))

			Expect(result.Launch.BOM).To(Equal(
				[]packit.BOMEntry{
					{
						Name: "poetry",
						Metadata: paketosbom.BOMMetadata{
							Checksum: paketosbom.BOMChecksum{
								Algorithm: paketosbom.SHA256,
								Hash:      "poetry-dependency-sha",
							},
							URI:     "poetry-dependency-uri",
							Version: "poetry-dependency-version",
						},
					},
				},
			))
		})
	})

	context("failure cases", func() {
		context("when the dependency cannot be resolved", func() {
			it.Before(func() {
				dependencyManager.ResolveCall.Returns.Error = errors.New("failed to resolve dependency")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError("failed to resolve dependency"))
			})
		})

		context("when the poetry layer cannot be retrieved", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(layersDir, "poetry.toml"), nil, 0000)
				Expect(err).NotTo(HaveOccurred())
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("failed to parse layer content metadata")))
			})
		})

		context("when the poetry layer cannot be reset", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(layersDir, "poetry", "something"), os.ModePerm)).To(Succeed())
				Expect(os.Chmod(filepath.Join(layersDir, "poetry"), 0500)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(filepath.Join(layersDir, "poetry"), os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("could not remove file")))
			})
		})

		context("when the install process returns an error", func() {
			it.Before(func() {
				installProcess.ExecuteCall.Returns.Error = errors.New("failed to run install process")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError("failed to run install process"))
			})
		})

		context("when generating the SBOM returns an error", func() {
			it.Before(func() {
				buildContext.BuildpackInfo.SBOMFormats = []string{"random-format"}
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(`unsupported SBOM format: 'random-format'`))
			})
		})

		context("when formatting the SBOM returns an error", func() {
			it.Before(func() {
				sbomGenerator.GenerateFromDependencyCall.Returns.Error = errors.New("failed to generate SBOM")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("failed to generate SBOM")))
			})
		})
	})
}
