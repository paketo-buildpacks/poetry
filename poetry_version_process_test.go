package poetry_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/poetry"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPoetryVersionProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		targetLayerPath string

		poetryVersionProcess poetry.PoetryVersionProcess
	)

	it.Before(func() {
		var err error
		targetLayerPath, err = os.MkdirTemp("", "targetLayerPath")
		Expect(err).NotTo(HaveOccurred())

		err = os.MkdirAll(filepath.Join(targetLayerPath, "bin"), os.ModePerm|os.ModeDir)
		Expect(err).NotTo(HaveOccurred())

		poetryVersionProcess = poetry.NewPoetryVersionProcess()
	})

	context("Execute", func() {
		context("the poetry executable exists", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(targetLayerPath, "bin", "poetry"), []byte("#!/usr/bin/env sh\necho Poetry version 99.88.77"), 0777)).To(Succeed())
			})

			it("returns the poetry version", func() {
				version, err := poetryVersionProcess.Execute(targetLayerPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(version).To(Equal("99.88.77"))
			})
		})

		context("failure cases", func() {
			context("the poetry executable does not exist", func() {
				it("returns an error", func() {
					_, err := poetryVersionProcess.Execute(targetLayerPath)
					Expect(err).NotTo(Succeed())
				})
			})
		})
	})
}
