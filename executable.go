package poetry

import (
	"github.com/paketo-buildpacks/packit/v2/pexec"
)

//go:generate faux --interface Executable --output fakes/executable.go

// Executable defines the interface for invoking an executable.
type Executable interface {
	Execute(pexec.Execution) error
}
