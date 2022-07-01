package poetry

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paketo-buildpacks/packit/v2/pexec"
)

type PoetryVersionProcess struct {
}

// NewPoetryVersionProcess creates a PoetryVersionProcess instance.
func NewPoetryVersionProcess() PoetryVersionProcess {
	return PoetryVersionProcess{}
}

// Execute returns the installed version of Poetry
func (p PoetryVersionProcess) Execute(targetLayerPath string) (string, error) {
	buffer := bytes.NewBuffer(nil)

	err := pexec.NewExecutable(filepath.Join(targetLayerPath, "bin", "poetry")).Execute(pexec.Execution{
		Args:   []string{"--version"},
		Env:    append(os.Environ(), fmt.Sprintf("PYTHONUSERBASE=%s", targetLayerPath)),
		Stdout: buffer,
		Stderr: buffer,
	})

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(strings.TrimPrefix(buffer.String(), "Poetry version ")), nil
}
