package poetry

const (
	PoetryVersion    = "poetry-version"
	PoetryDependency = "poetry"
	PoetryLayerName  = "poetry"
	CPython          = "cpython"
	Pip              = "pip"
	DefaultVersions  = "default-versions"
)

var Priorities = []interface{}{
	"BP_POETRY_VERSION",
	DefaultVersions,
}
