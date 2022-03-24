package fakes

import "sync"

type PyProjectPythonVersionParser struct {
	ParsePythonVersionCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			String string
		}
		Returns struct {
			String string
			Error  error
		}
		Stub func(string) (string, error)
	}
}

func (f *PyProjectPythonVersionParser) ParsePythonVersion(param1 string) (string, error) {
	f.ParsePythonVersionCall.mutex.Lock()
	defer f.ParsePythonVersionCall.mutex.Unlock()
	f.ParsePythonVersionCall.CallCount++
	f.ParsePythonVersionCall.Receives.String = param1
	if f.ParsePythonVersionCall.Stub != nil {
		return f.ParsePythonVersionCall.Stub(param1)
	}
	return f.ParsePythonVersionCall.Returns.String, f.ParsePythonVersionCall.Returns.Error
}
