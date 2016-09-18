package oldaws

import (
	"errors"

	"koding/kites/kloud/basestack"
)

type Meta struct {
	AlwaysOn         bool   `bson:"alwaysOn"`
	InstanceID       string `structs:"instanceId" bson:"instanceId"`
	AvailabilityZone string `structs:"availabilityZone" bson:"availabilityZone"`
	PlacementGroup   string `structs:"placementGroup" bson:"placementGroup"`
	Region           string `structs:"region" bson:"region"`
	StorageSize      int    `structs:"storage_size" bson:"storage_size"`
}

func (mt *Meta) Valid() error {
	if mt.Region == "" {
		return errors.New("invalid empty region")
	}

	return nil
}

// Machine represents a single MongodDB document from the jMachines
// collection.
type Machine struct {
	*basestack.BaseMachine
}

var _ basestack.Machine = (*Machine)(nil)

func (m *Machine) Credential() *Credential {
	return m.BaseMachine.Credential.(*Credential)
}

func (m *Machine) Bootstrap() *Bootstrap {
	return m.BaseMachine.Bootstrap.(*Bootstrap)
}

func (m *Machine) Metadata() *Meta {
	return m.BaseMachine.Metadata.(*Meta)
}
