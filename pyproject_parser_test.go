package poetry_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/poetry"
	"github.com/sclevine/spec"
)

func testPyProjectParser(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir    string
		pyProjectToml string

		parser poetry.PyProjectParser
	)

	const (
		version = `[tool.poetry.dependencies]
python = "1.2.3"`
		version_pep621 = `[project]
requires-python = ">=1.2.3"`
		exact_version_pep621 = `[project]
requires-python = "==1.2.3"`
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		pyProjectToml = filepath.Join(workingDir, poetry.PyProjectTomlFile)

		parser = poetry.NewPyProjectParser()
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("Calling ParsePythonVersion", func() {
		it("parses version", func() {
			Expect(os.WriteFile(pyProjectToml, []byte(version), 0644)).To(Succeed())

			version, err := parser.ParsePythonVersion(pyProjectToml)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("1.2.3"))
		})

		it("parses version PEP621", func() {
			Expect(os.WriteFile(pyProjectToml, []byte(version_pep621), 0644)).To(Succeed())

			version, err := parser.ParsePythonVersion(pyProjectToml)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(">=1.2.3"))
		})

		it("parses exact version PEP621", func() {
			Expect(os.WriteFile(pyProjectToml, []byte(exact_version_pep621), 0644)).To(Succeed())

			version, err := parser.ParsePythonVersion(pyProjectToml)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("1.2.3"))
		})

		it("returns empty string if file does not contain 'tool.poetry.dependencies.python' or project.requires-python", func() {
			Expect(os.WriteFile(pyProjectToml, []byte(""), 0644)).To(Succeed())

			version, err := parser.ParsePythonVersion(pyProjectToml)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(""))
		})

		context("error handling", func() {
			it("fails if file does not exist", func() {
				_, err := parser.ParsePythonVersion("not-a-valid-dir")
				Expect(err).To(HaveOccurred())
			})
		})
	})
}
