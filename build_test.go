package poetry_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	packit "github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"

	//nolint Ignore SA1019, informed usage of deprecated package
	"github.com/paketo-buildpacks/packit/v2/paketosbom"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/poetry"
	"github.com/paketo-buildpacks/poetry/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir         string
		cnbDir            string
		timestamp         time.Time
		entryResolver     *fakes.EntryResolver
		dependencyManager *fakes.DependencyManager
		installProcess    *fakes.InstallProcess
		siteProcess       *fakes.SitePackageProcess
		buffer            *bytes.Buffer

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		timestamp = time.Now()
		clock := chronos.NewClock(func() time.Time {
			return timestamp
		})

		entryResolver = &fakes.EntryResolver{}
		entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
			Name: "poetry",
		}

		dependencyManager = &fakes.DependencyManager{}
		dependencyManager.ResolveCall.Returns.Dependency = postal.Dependency{
			ID:      "poetry",
			Name:    "poetry-dependency-name",
			SHA256:  "poetry-dependency-sha",
			Stacks:  []string{"some-stack"},
			URI:     "poetry-dependency-uri",
			Version: "poetry-dependency-version",
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

		installProcess = &fakes.InstallProcess{}
		siteProcess = &fakes.SitePackageProcess{}
		siteProcess.ExecuteCall.Returns.String = filepath.Join(layersDir, "poetry", "lib", "python3.8", "site-packages")

		buffer = bytes.NewBuffer(nil)
		logEmitter := scribe.NewEmitter(buffer)

		build = poetry.Build(entryResolver, dependencyManager, installProcess, siteProcess, clock, logEmitter)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
	})

	it("returns a result that installs poetry", func() {
		result, err := build(packit.BuildContext{
			BuildpackInfo: packit.BuildpackInfo{
				Name:    "Some Buildpack",
				Version: "some-version",
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
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(result).To(Equal(packit.BuildResult{
			Layers: []packit.Layer{
				{
					Name: "poetry",
					Path: filepath.Join(layersDir, "poetry"),
					SharedEnv: packit.Environment{
						"PYTHONPATH.delim":   ":",
						"PYTHONPATH.prepend": filepath.Join(layersDir, "poetry", "lib/python3.8/site-packages"),
					},
					BuildEnv:         packit.Environment{},
					LaunchEnv:        packit.Environment{},
					ProcessLaunchEnv: map[string]packit.Environment{},
					Build:            false,
					Launch:           false,
					Cache:            false,
					Metadata: map[string]interface{}{
						"dependency-sha": "poetry-dependency-sha",
						"built_at":       timestamp.Format(time.RFC3339Nano),
					},
				},
			},
		}))

		Expect(entryResolver.ResolveCall.Receives.Name).To(Equal("poetry"))
		Expect(entryResolver.ResolveCall.Receives.Entries).To(Equal([]packit.BuildpackPlanEntry{
			{Name: "poetry"},
		}))

		Expect(entryResolver.MergeLayerTypesCall.Receives.Name).To(Equal("poetry"))
		Expect(entryResolver.MergeLayerTypesCall.Receives.Entries).To(Equal([]packit.BuildpackPlanEntry{
			{Name: "poetry"},
		}))

		Expect(dependencyManager.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
		Expect(dependencyManager.ResolveCall.Receives.Id).To(Equal("poetry"))
		Expect(dependencyManager.ResolveCall.Receives.Version).To(Equal("default"))
		Expect(dependencyManager.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyManager.DeliverCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:      "poetry",
			Name:    "poetry-dependency-name",
			SHA256:  "poetry-dependency-sha",
			Stacks:  []string{"some-stack"},
			URI:     "poetry-dependency-uri",
			Version: "poetry-dependency-version",
		}))
		Expect(dependencyManager.DeliverCall.Receives.CnbPath).To(Equal(cnbDir))
		Expect(dependencyManager.DeliverCall.Receives.DestinationPath).To(ContainSubstring("poetry-source"))
		Expect(dependencyManager.DeliverCall.Receives.PlatformPath).To(Equal("platform"))

		Expect(dependencyManager.GenerateBillOfMaterialsCall.CallCount).To(Equal(1))
		Expect(dependencyManager.GenerateBillOfMaterialsCall.Receives.Dependencies).To(Equal([]postal.Dependency{
			{
				ID:      "poetry",
				Name:    "poetry-dependency-name",
				SHA256:  "poetry-dependency-sha",
				Stacks:  []string{"some-stack"},
				URI:     "poetry-dependency-uri",
				Version: "poetry-dependency-version",
			},
		}))

		Expect(installProcess.ExecuteCall.Receives.SrcPath).To(Equal(dependencyManager.DeliverCall.Receives.DestinationPath))
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
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			entryResolver.MergeLayerTypesCall.Returns.Build = true
		})

		it("makes the layer available in those phases", func() {
			result, err := build(packit.BuildContext{
				CNBPath: cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "poetry",
							Metadata: map[string]interface{}{
								"build":  true,
								"launch": true,
							},
						},
					},
				},
				Layers: packit.Layers{Path: layersDir},
				Stack:  "some-stack",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Name: "poetry",
						Path: filepath.Join(layersDir, "poetry"),
						SharedEnv: packit.Environment{
							"PYTHONPATH.delim":   ":",
							"PYTHONPATH.prepend": filepath.Join(layersDir, "poetry", "lib/python3.8/site-packages"),
						},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            true,
						Launch:           true,
						Cache:            true,
						Metadata: map[string]interface{}{
							"dependency-sha": "poetry-dependency-sha",
							"built_at":       timestamp.Format(time.RFC3339Nano),
						},
					},
				},
				Build: packit.BuildMetadata{
					BOM: []packit.BOMEntry{
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
				},
				Launch: packit.LaunchMetadata{
					BOM: []packit.BOMEntry{
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
				},
			}))
		})
	})

	context("failure cases", func() {
		context("when the dependency cannot be resolved", func() {
			it.Before(func() {
				dependencyManager.ResolveCall.Returns.Error = errors.New("failed to resolve dependency")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "poetry"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError("failed to resolve dependency"))
			})
		})

		context("when the poetry layer cannot be retrieved", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(layersDir, "poetry.toml"), nil, 0000)
				Expect(err).NotTo(HaveOccurred())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "poetry"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
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
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "poetry"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError(ContainSubstring("could not remove file")))
			})
		})

		context("when the dependency cannot be installed", func() {
			it.Before(func() {
				dependencyManager.DeliverCall.Returns.Error = errors.New("failed to deliver dependency")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "poetry"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError("failed to deliver dependency"))
			})
		})
	})
}
