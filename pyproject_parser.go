package poetry

import (
	"github.com/BurntSushi/toml"
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

type PyProjectParser struct {
}

func NewPyProjectParser() PyProjectParser {
	return PyProjectParser{}
}

func (p PyProjectParser) ParsePythonVersion(pyProjectToml string) (string, error) {
	var pyProject PyProjectToml

	_, err := toml.DecodeFile(pyProjectToml, &pyProject)
	if err != nil {
		return "", err
	}

	return pyProject.Tool.Poetry.Dependencies.Python, nil
}
