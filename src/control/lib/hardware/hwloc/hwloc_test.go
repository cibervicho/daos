//
// (C) Copyright 2021 Intel Corporation.
//
// SPDX-License-Identifier: BSD-2-Clause-Patent
//

package hwloc

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/daos-stack/daos/src/control/common"
	"github.com/daos-stack/daos/src/control/lib/hardware"
	"github.com/daos-stack/daos/src/control/logging"
)

func TestHwlocProvider_GetTopology(t *testing.T) {
	for name, tc := range map[string]struct {
		hwlocXMLFile string
		expResult    *hardware.Topology
	}{
		"boro-84": {
			hwlocXMLFile: "testdata/boro-84.xml",
			expResult: &hardware.Topology{
				NUMANodes: map[uint]*hardware.NUMANode{
					0: {
						ID:       0,
						NumCores: 24,
						PCIDevices: map[string][]*hardware.Device{
							"0000:18:00.0": {
								{
									Name:    "ib0",
									Type:    hardware.DeviceTypeNetwork,
									PCIAddr: "0000:18:00.0",
								},
								{
									Name:    "hfi1_0",
									Type:    hardware.DeviceTypeOpenFabrics,
									PCIAddr: "0000:18:00.0",
								},
							},
							"0000:3d:00.1": {
								{
									Name:    "eth0",
									Type:    hardware.DeviceTypeNetwork,
									PCIAddr: "0000:3d:00.1",
								},
								{
									Name:    "i40iw0",
									Type:    hardware.DeviceTypeOpenFabrics,
									PCIAddr: "0000:3d:00.1",
								},
							},
						},
					},
					1: {
						ID:         1,
						NumCores:   24,
						PCIDevices: map[string][]*hardware.Device{},
					},
				},
			},
		},
		"wolf-133": {
			hwlocXMLFile: "testdata/wolf-133.xml",
			expResult: &hardware.Topology{
				NUMANodes: map[uint]*hardware.NUMANode{
					0: {
						ID:       0,
						NumCores: 24,
						PCIDevices: map[string][]*hardware.Device{
							"0000:18:00.0": {
								{
									Name:    "ib0",
									Type:    hardware.DeviceTypeNetwork,
									PCIAddr: "0000:18:00.0",
								},
								{
									Name:    "hfi1_0",
									Type:    hardware.DeviceTypeOpenFabrics,
									PCIAddr: "0000:18:00.0",
								},
							},
							"0000:3d:00.1": {
								{
									Name:    "eth0",
									Type:    hardware.DeviceTypeNetwork,
									PCIAddr: "0000:3d:00.1",
								},
								{
									Name:    "i40iw0",
									Type:    hardware.DeviceTypeOpenFabrics,
									PCIAddr: "0000:3d:00.1",
								},
							},
						},
					},
					1: {
						ID:         1,
						NumCores:   24,
						PCIDevices: map[string][]*hardware.Device{},
					},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			log, buf := logging.NewTestLogger(name)
			defer common.ShowBufferOnFailure(t, buf)

			_, err := os.Stat(tc.hwlocXMLFile)
			common.AssertEqual(t, err, nil, "unable to read hwloc test topology file")
			os.Setenv("HWLOC_XMLFILE", tc.hwlocXMLFile)
			defer os.Unsetenv("HWLOC_XMLFILE")

			// cmd := exec.Command("hwloc-ls")
			// cmd.Stdout = os.Stdout
			// cmd.Run()

			provider := NewProvider(log)
			result, err := provider.GetTopology()
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.expResult, result); diff != "" {
				t.Errorf("(-want, +got)\n%s\n", diff)
			}
		})

	}
}
