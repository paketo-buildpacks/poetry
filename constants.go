package poetry

const (
	DependencyChecksumKey = "dependency-checksum"
	PoetryDependency      = "poetry"
	PoetryLayerName       = "poetry"
	CPython               = "cpython"
	Pip                   = "pip"
)

var Priorities = []interface{}{
	"BP_PIP_VERSION",
}
