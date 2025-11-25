package poetry

import (
	"strings"
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
	Project struct {
		RequiresPython string `toml:"requires-python"`
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

	if pyProject.Project.RequiresPython != "" {
		return strings.Trim(pyProject.Project.RequiresPython, "="), nil
	}
	return strings.Trim(pyProject.Tool.Poetry.Dependencies.Python, "="), nil
}
