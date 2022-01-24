package fakes

import "sync"

type InstallProcess struct {
	ExecuteCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			SrcPath         string
			TargetLayerPath string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string) error
	}
}

func (f *InstallProcess) Execute(param1 string, param2 string) error {
	f.ExecuteCall.mutex.Lock()
	defer f.ExecuteCall.mutex.Unlock()
	f.ExecuteCall.CallCount++
	f.ExecuteCall.Receives.SrcPath = param1
	f.ExecuteCall.Receives.TargetLayerPath = param2
	if f.ExecuteCall.Stub != nil {
		return f.ExecuteCall.Stub(param1, param2)
	}
	return f.ExecuteCall.Returns.Error
}
