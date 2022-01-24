package poetry

const (
	DependencySHAKey = "dependency-sha"
	PoetryDependency = "poetry"
	PoetryLayerName  = "poetry"
	CPython          = "cpython"
	Pip              = "pip"
)

var Priorities = []interface{}{
	"BP_PIP_VERSION",
}

