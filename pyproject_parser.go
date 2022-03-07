package poetry

import (
	"errors"
	"github.com/BurntSushi/toml"
	"path/filepath"
)

type PyProjectToml struct {
	Tool struct {
		Poetry struct {
			Dependencies struct {
				Python string
			}
		}
	}
}

const PyProjectTomlFile = "pyproject.toml"

type PyProjectParser struct {
}

func NewPyProjectParser() PyProjectParser {
	return PyProjectParser{}
}

func (p PyProjectParser) ParsePythonVersion(path string) (version string, err error) {
	var pyProject PyProjectToml
	_, err = toml.DecodeFile(filepath.Join(path, PyProjectTomlFile), &pyProject)

	if pyProject.Tool.Poetry.Dependencies.Python == "" {
		return "", errors.New("pyproject.toml must include [tool.poetry.dependencies.python], see https://python-poetry.org/docs/pyproject/#dependencies-and-dev-dependencies")
	}
	return pyProject.Tool.Poetry.Dependencies.Python, err
}
