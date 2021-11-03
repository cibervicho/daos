//
// (C) Copyright 2021 Intel Corporation.
//
// SPDX-License-Identifier: BSD-2-Clause-Patent
//

// hwloc is a set of Go bindings for interacting with the hwloc library.
package hwloc

import (
	"github.com/daos-stack/daos/src/control/lib/hardware"
	"github.com/pkg/errors"
)

type Provider struct {
	api *api
}

func (p *Provider) GetTopology() (*hardware.Topology, error) {
	topo, cleanup, err := getTopo(p.api)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	nodes, err := p.getNUMANodes(topo)
	if err != nil {
		return nil, err
	}

	return &hardware.Topology{
		NUMANodes: nodes,
	}, nil
}

func (p *Provider) getNUMANodes(topo *topology) (map[uint]*hardware.NUMANode, error) {
	coresByNode := p.getCoreCountsPerNodeSet(topo)
	devsByNode := p.getDevicesPerNodeSet(topo)

	nodes := make(map[uint]*hardware.NUMANode)

	prevNode := (*object)(nil)
	for {
		numaObj, err := topo.GetNextObjByType(ObjTypeNUMANode, prevNode)
		if err != nil {
			break
		}

		nodeStr := numaObj.NodeSet().String()

		newNode := &hardware.NUMANode{
			ID:       numaObj.OSIndex(),
			NumCores: coresByNode[nodeStr],
			Devices:  devsByNode[nodeStr],
		}

		nodes[newNode.ID] = newNode

		prevNode = numaObj
	}

	return nodes, nil
}

func (p *Provider) getCoreCountsPerNodeSet(topo *topology) map[string]uint {
	prevCore := (*object)(nil)
	coresPerNode := make(map[string]uint)
	for {
		coreObj, err := topo.GetNextObjByType(ObjTypeCore, prevCore)
		if err != nil {
			break
		}

		coresPerNode[coreObj.NodeSet().String()]++

		prevCore = coreObj
	}

	return coresPerNode
}

func (p *Provider) getDevicesPerNodeSet(topo *topology) map[string]map[string]*hardware.Device {
	prevDev := (*object)(nil)
	devicesPerNode := make(map[string]map[string]*hardware.Device)
	for {
		devObj, err := topo.GetNextObjByType(ObjTypeOSDevice, prevDev)
		if err != nil {
			break
		}

		key := devObj.NodeSet().String()
		devicesPerNode[key][devObj.Name()] = &hardware.Device{
			Name: devObj.Name(),
		}

		prevDev = devObj
	}

	return devicesPerNode
}

// // API is an interface for basic API operations, including synchronizing access.
// type API interface {
// 	Lock()
// 	Unlock()

// 	runtimeVersion() uint
// 	compiledVersion() uint
// 	newTopology() (internalTopology, func(), error)
// }

// type internalTopology interface {
// 	Topology
// 	load() error
// 	setFlags() error
// }

// type ObjType uint

// // Topology is an interface for an hwloc topology.
// type Topology interface {
// 	// GetProcessCPUSet gets a CPUSet associated with a given pid, and returns the CPUSet and its
// 	// cleanup function, or an error.
// 	GetProcessCPUSet(pid int32, flags int) (CPUSet, func(), error)
// 	// GetObjByDepth gets an Object at a given depth and index in the topology.
// 	GetObjByDepth(depth int, index uint) (Object, error)
// 	// GetTypeDepth fetches the depth of a given ObjType in the topology.
// 	GetTypeDepth(objType ObjType) int
// 	// GetNumObjAtDepth fetches the number of objects located at a given depth in the topology.
// 	GetNumObjAtDepth(depth int) uint
// }

// // Object is an interface for an object in a Topology.
// type Object interface {
// 	// Name returns the name of the object.
// 	Name() string
// 	// LogicalIndex returns the logical index of the object.
// 	LogicalIndex() uint
// 	// GetNumSiblings returns the number of siblings this object has in the topology.
// 	GetNumSiblings() uint
// 	// GetChild gets the object's child with a given index, or returns an error.
// 	GetChild(index uint) (Object, error)
// 	// GetAncestorByType gets the object's ancestor of a given type, or returns an error.
// 	GetAncestorByType(objType ObjType) (Object, error)
// 	// GetNonIOAncestor gets the object's non-IO ancestor, if any, or returns an error.
// 	GetNonIOAncestor() (Object, error)
// 	// CPUSet gets the CPUSet associated with the object.
// 	CPUSet() CPUSet
// 	// NodeSet gets the NodeSet associated with the object.
// 	NodeSet() NodeSet
// }

// // CPUSet is an interface for a CPU set.
// type CPUSet interface {
// 	String() string
// 	// Intersects determines if this CPUSet intersects with another one.
// 	Intersects(CPUSet) bool
// 	// IsSubsetOf determines if this CPUSet is included completely within another one.
// 	IsSubsetOf(CPUSet) bool
// 	// ToNodeSet translates this CPUSet into a NodeSet and returns it with its cleanup function,
// 	// or returns an error.
// 	ToNodeSet() (NodeSet, func(), error)
// }

// // NodeSet is an interface for a node set.
// type NodeSet interface {
// 	String() string
// 	// Intersects determines if this NodeSet intersects with another one.
// 	Intersects(NodeSet) bool
// }

// // GetAPI fetches an API reference.
// func GetAPI() API {
// 	return &api{}
// }

// getTopo initializes the hwloc topology and returns it to the caller along with the topology
// cleanup function.
func getTopo(api *api) (*topology, func(), error) {
	if api == nil {
		return nil, nil, errors.New("nil API")
	}

	if err := checkVersion(api); err != nil {
		return nil, nil, err
	}

	api.Lock()
	defer api.Unlock()

	topo, cleanup, err := api.newTopology()
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err != nil {
			cleanup()
		}
	}()

	if err = topo.setFlags(); err != nil {
		return nil, nil, err
	}

	if err = topo.load(); err != nil {
		return nil, nil, err
	}

	return topo, cleanup, nil
}

// checkVersion verifies the runtime API is compatible with the compiled version of the API.
func checkVersion(api *api) error {
	version := api.runtimeVersion()
	compVersion := api.compiledVersion()
	if (version >> 16) != (compVersion >> 16) {
		return errors.Errorf("hwloc API incompatible with runtime: compiled for version 0x%x but using 0x%x\n",
			compVersion, version)
	}
	return nil
}
