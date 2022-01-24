package poetry_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-community/poetry"
	"github.com/paketo-community/poetry/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPoetryInstallProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		srcPath       string
		destLayerPath string
		executable    *fakes.Executable

		poetryInstallProcess poetry.PoetryInstallProcess
	)

	it.Before(func() {
		var err error
		srcPath, err = os.MkdirTemp("", "poetry-source")
		Expect(err).NotTo(HaveOccurred())

		destLayerPath, err = os.MkdirTemp("", "poetry")
		Expect(err).NotTo(HaveOccurred())

		executable = &fakes.Executable{}

		poetryInstallProcess = poetry.NewPoetryInstallProcess(executable)
	})

	context("Execute", func() {
		context("there is a poetry dependency to install", func() {
			it("installs it to the poetry layer", func() {
				err := poetryInstallProcess.Execute(srcPath, destLayerPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.Receives.Execution.Env).To(Equal(append(os.Environ(), fmt.Sprintf("PYTHONUSERBASE=%s", destLayerPath))))
				Expect(executable.ExecuteCall.Receives.Execution.Args).To(Equal([]string{"install", "poetry", "--user", fmt.Sprintf("--find-links=%s", srcPath)}))
			})
		})

		context("failure cases", func() {
			context("the install process fails", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						fmt.Fprintln(execution.Stdout, "stdout output")
						fmt.Fprintln(execution.Stderr, "stderr output")
						return errors.New("installing poetry failed")
					}
				})

				it("returns an error", func() {
					err := poetryInstallProcess.Execute(srcPath, destLayerPath)
					Expect(err).To(MatchError(ContainSubstring("installing poetry failed")))
					Expect(err).To(MatchError(ContainSubstring("stdout output")))
					Expect(err).To(MatchError(ContainSubstring("stderr output")))
				})
			})
		})
	})
}
