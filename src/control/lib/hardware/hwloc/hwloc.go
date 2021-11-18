//
// (C) Copyright 2021 Intel Corporation.
//
// SPDX-License-Identifier: BSD-2-Clause-Patent
//

// hwloc is a set of Go bindings for interacting with the hwloc library.
package hwloc

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/daos-stack/daos/src/control/lib/hardware"
	"github.com/daos-stack/daos/src/control/logging"
)

// NewProvider returns a new hwloc Provider.
func NewProvider(log logging.Logger) hardware.TopologyProvider {
	return &Provider{
		api: &api{},
		log: log,
	}
}

type Provider struct {
	api *api
	log logging.Logger
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

	nodes := make(map[uint]*hardware.NUMANode)

	prevNode := (*object)(nil)
	for {
		numaObj, err := topo.getNextObjByType(objTypeNUMANode, prevNode)
		if err != nil {
			break
		}

		nodeStr := numaObj.nodeSet().String()

		devs, err := p.getPCIDevsForNUMANode(numaObj)
		if err != nil {
			return nil, err
		}

		newNode := &hardware.NUMANode{
			ID:         numaObj.osIndex(),
			NumCores:   coresByNode[nodeStr],
			PCIDevices: devs,
		}

		nodes[newNode.ID] = newNode

		prevNode = numaObj
	}

	// if len(nodes) == 0 {
	// 	nodes[0] = &hardware.NUMANode{
	// 		ID: 0,
	// 	}
	// }

	return nodes, nil
}

func (p *Provider) getCoreCountsPerNodeSet(topo *topology) map[string]uint {
	prevCore := (*object)(nil)
	coresPerNode := make(map[string]uint)
	for {
		coreObj, err := topo.getNextObjByType(objTypeCore, prevCore)
		if err != nil {
			break
		}

		coresPerNode[coreObj.nodeSet().String()]++

		prevCore = coreObj
	}

	return coresPerNode
}

func (p *Provider) getPCIDevsForNUMANode(numaNode *object) (map[string][]*hardware.Device, error) {
	pciDevs := make(map[string][]*hardware.Device)

	p.addPCIDevsBelowObj(numaNode, pciDevs)

	return pciDevs, nil
}

func (p *Provider) addPCIDevsBelowObj(obj *object, pciDevs map[string][]*hardware.Device) {
	fmt.Printf("num children = %d\n", obj.getNumChildren())
	for i := uint(0); i < obj.getNumChildren(); i++ {
		cur, err := obj.getChild(i)
		if err != nil {
			p.log.Error(err.Error())
			continue
		}

		if cur.objType() == objTypePCIDevice {
			addr, err := cur.pciAddr()
			if err != nil {
				panic(err)
			}

			for j := uint(0); j < cur.getNumChildren(); j++ {
				dev, err := cur.getChild(j)
				if err != nil {
					p.log.Error(err.Error())
					break
				}

				if dev.objType() != objTypeOSDevice {
					p.log.Debugf("skipping object type %d", dev.objType())
					continue
				}

				osDevType, err := dev.osDevType()
				if err != nil {
					p.log.Error(err.Error())
					continue
				}
				switch osDevType {
				case osDevTypeNetwork, osDevTypeOpenFabrics:
					pciDevs[addr] = append(pciDevs[addr], &hardware.Device{
						Name:    dev.name(),
						Type:    osDevTypeToHardwareDevType(osDevType),
						PCIAddr: addr,
					})
				}
			}
		}

		p.addPCIDevsBelowObj(cur, pciDevs)
	}
}

func osDevTypeToHardwareDevType(osType int) hardware.DeviceType {
	switch osType {
	case osDevTypeNetwork:
		return hardware.DeviceTypeNetwork
	case osDevTypeOpenFabrics:
		return hardware.DeviceTypeOpenFabrics
	}

	return hardware.DeviceTypeUnknown
}

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
