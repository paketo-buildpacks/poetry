package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testVersions(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("when the buildpack is run with pack build", func() {
		var (
			name   string
			source string

			containersMap map[string]interface{}
			imagesMap     map[string]interface{}
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())

			containersMap = map[string]interface{}{}
			imagesMap = map[string]interface{}{}

			source, err = occam.Source(filepath.Join("testdata", "default_app"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			for containerID := range containersMap {
				Expect(docker.Container.Remove.Execute(containerID)).To(Succeed())
			}
			for imageID := range imagesMap {
				Expect(docker.Image.Remove.Execute(imageID)).To(Succeed())
			}
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("builds and runs successfully with multiple provided dependency versions", func() {
			var err error

			firstPoetryVersion := buildpackInfo.Metadata.Dependencies[0].Version
			secondPoetryVersion := buildpackInfo.Metadata.Dependencies[1].Version

			Expect(firstPoetryVersion).NotTo(Equal(secondPoetryVersion))

			firstImage, firstLogs, err := pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.CPython.Online,
					settings.Buildpacks.Pip.Online,
					settings.Buildpacks.Poetry.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				WithEnv(map[string]string{"BP_POETRY_VERSION": firstPoetryVersion}).
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), firstLogs.String)

			imagesMap[firstImage.ID] = nil

			Expect(firstLogs).To(ContainLines(
				ContainSubstring(fmt.Sprintf(`Selected Poetry version (using BP_POETRY_VERSION): %s`, firstPoetryVersion)),
			))

			firstContainer, err := docker.Container.Run.
				WithCommand("poetry --version").
				Execute(firstImage.ID)
			Expect(err).ToNot(HaveOccurred())

			containersMap[firstContainer.ID] = nil

			Eventually(func() string {
				cLogs, err := docker.Container.Logs.Execute(firstContainer.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).Should(MatchRegexp(fmt.Sprintf(`Poetry.*version %s`, firstPoetryVersion)))

			secondImage, secondLogs, err := pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.CPython.Online,
					settings.Buildpacks.Pip.Online,
					settings.Buildpacks.Poetry.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				WithEnv(map[string]string{"BP_POETRY_VERSION": secondPoetryVersion}).
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), secondLogs.String)

			imagesMap[secondImage.ID] = nil

			Expect(secondLogs).To(ContainLines(
				ContainSubstring(fmt.Sprintf(`Selected Poetry version (using BP_POETRY_VERSION): %s`, secondPoetryVersion)),
			))

			secondContainer, err := docker.Container.Run.
				WithCommand("poetry --version").
				Execute(secondImage.ID)
			Expect(err).ToNot(HaveOccurred())

			containersMap[secondContainer.ID] = nil

			Eventually(func() string {
				cLogs, err := docker.Container.Logs.Execute(secondContainer.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).Should(MatchRegexp(fmt.Sprintf(`Poetry.*version %s`, secondPoetryVersion)))
		})
	})
}
