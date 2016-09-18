package oldaws

import "koding/kites/kloud/basestack"

func init() {
	p := &basestack.Provider{
		Name:          "aws",
		ResourceName:  "instance",
		NewMachine:    nil,
		NewStack:      nil,
		NewCredential: func() interface{} { return &Credential{} },
		NewBootstrap:  func() interface{} { return &Bootstrap{} },
		NewMetadata:   func() interface{} { return &Metadata{} },
	}

	basestack.Register(p)
}
