//
// (C) Copyright 2021 Intel Corporation.
//
// SPDX-License-Identifier: BSD-2-Clause-Patent
//

// hardware is a package that reveals details about the hardware topology.
package hardware

// Topology is a hierarchy of hardware devices grouped under NUMA nodes.
type Topology struct {
	NUMANodes map[uint]*NUMANode
}

// NUMANode represents an individual NUMA node in the system and the devices associated with it.
type NUMANode struct {
	ID         uint
	NumCores   uint
	PCIDevices map[string][]*Device
}

// DeviceType indicates the type of a hardware device.
type DeviceType uint

const (
	// DeviceTypeUnknown indicates a device type that is not recognized.
	DeviceTypeUnknown DeviceType = iota
	// DeviceTypeNetwork indicates a standard network device.
	DeviceTypeNetwork
	// DeviceTypeOpenFabrics indicates an OpenFabrics device.
	DeviceTypeOpenFabrics
)

func (t DeviceType) String() string {
	switch t {
	case DeviceTypeNetwork:
		return "Network"
	case DeviceTypeOpenFabrics:
		return "OpenFabrics"
	}

	return "Unknown"
}

// Device represents an individual hardware device.
type Device struct {
	Name    string
	Type    DeviceType
	PCIAddr string
}

// TopologyProvider is an interface for acquiring a system topology.
type TopologyProvider interface {
	GetTopology() (*Topology, error)
}
