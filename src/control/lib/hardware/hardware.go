//
// (C) Copyright 2021 Intel Corporation.
//
// SPDX-License-Identifier: BSD-2-Clause-Patent
//

// hardware is a package that reveals details about the hardware topology.
package hardware

// type NUMAProvider interface {
// 	NumNUMANodes() uint
// 	NumCoresPerNUMANode() uint
// 	IsNUMAAware() bool
// 	NUMANodeForPID(int32) (int, error)
// 	VerifyDeviceIsNUMALocal(dev string, numaNode uint) error
// }

type Topology struct {
	NUMANodes map[uint]*NUMANode
}

type NUMANode struct {
	ID       uint
	NumCores uint
	Devices  map[string]*Device
}

type Device struct {
	Name string
}

type FabricInterface struct {
	// Provider    string `json:"provider"`
	Name     string `json:"device"`
	Domain   string `json:"domain"`
	NUMANode uint   `json:"numanode"`
	// Priority    int    `json:"priority"`
	// NetDevClass uint32 `json:"netdevclass"`
}

type TopologyProvider interface {
	GetTopology() (*Topology, error)
}
