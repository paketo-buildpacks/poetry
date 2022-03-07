package poetry_test

import (
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/poetry"
	"github.com/sclevine/spec"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func testPyProjectParser(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		parser     poetry.PyProjectParser
	)

	const (
		version1 = `[tool.poetry.dependencies]
python = "===1.2.3"`
		version2 = `[tool.poetry.dependencies]
python = "abcd"`
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		parser = poetry.NewPyProjectParser()
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("Calling ParseVersion", func() {
		it("parses version1", func() {
			Expect(ioutil.WriteFile(
				filepath.Join(workingDir, poetry.PyProjectTomlFile),
				[]byte(version1), 0644)).To(Succeed())

			version, err := parser.ParsePythonVersion(workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("===1.2.3"))
		})

		it("parses version2", func() {
			Expect(ioutil.WriteFile(
				filepath.Join(workingDir, poetry.PyProjectTomlFile),
				[]byte(version2), 0644)).To(Succeed())

			version, err := parser.ParsePythonVersion(workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("abcd"))
		})

		it("returns error if file does not exist", func() {
			_, err := parser.ParsePythonVersion("not a valid dir")
			Expect(err).To(HaveOccurred())
		})

		// the python dependency is mandatory
		// https://python-poetry.org/docs/pyproject/#dependencies-and-dev-dependencies
		it("returns error if file does not contain 'tool.poetry.dependencies.python'", func() {
			Expect(ioutil.WriteFile(
				filepath.Join(workingDir, poetry.PyProjectTomlFile),
				[]byte(""), 0644)).To(Succeed())

			_, err := parser.ParsePythonVersion(workingDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("pyproject.toml must include [tool.poetry.dependencies.python], see https://python-poetry.org/docs/pyproject/#dependencies-and-dev-dependencies"))
		})
	})
}
