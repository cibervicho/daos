//
// (C) Copyright 2021 Intel Corporation.
//
// SPDX-License-Identifier: BSD-2-Clause-Patent
//

package hwloc

// func TestHwloc_GetAPI(t *testing.T) {
// 	if GetAPI() == nil {
// 		t.Fatal("GetAPI() returned nil")
// 	}
// }

// type mockAPI struct {
// 	runtimeVer     uint
// 	compVer        uint
// 	numCallsLock   uint
// 	numCallsUnlock uint
// 	newTopo        internalTopology
// 	newTopoCleanup func()
// 	newTopoErr     error
// }

// func (m *mockAPI) Lock() {
// 	m.numCallsLock++
// }

// func (m *mockAPI) Unlock() {
// 	m.numCallsUnlock++
// }

// func (m *mockAPI) runtimeVersion() uint {
// 	return m.runtimeVer
// }

// func (m *mockAPI) compiledVersion() uint {
// 	return m.compVer
// }

// func (m *mockAPI) newTopology() (internalTopology, func(), error) {
// 	return m.newTopo, m.newTopoCleanup, m.newTopoErr
// }

// type mockTopology struct {
// 	setFlagsErr error
// 	loadErr     error
// }

// func (m *mockTopology) GetProcessCPUSet(pid int32, flags int) (CPUSet, func(), error) {
// 	return nil, nil, nil
// }

// func (m *mockTopology) setFlags() error {
// 	return m.setFlagsErr
// }

// func (m *mockTopology) load() error {
// 	return m.loadErr
// }

// func (m *mockTopology) GetObjByDepth(depth int, index uint) (Object, error) {
// 	return nil, nil
// }

// func (m *mockTopology) GetTypeDepth(objType ObjType) int {
// 	return 0
// }

// func (m *mockTopology) GetNumObjAtDepth(depth int) uint {
// 	return 0
// }

// func TestHwloc_GetTopology(t *testing.T) {
// 	var testCleanupCalled uint
// 	testCleanup := func() {
// 		testCleanupCalled++
// 	}

// 	for name, tc := range map[string]struct {
// 		api                   API
// 		expTimesLocked        uint
// 		expTimesCleanupCalled uint
// 		expTopology           Topology
// 		expErr                error
// 	}{
// 		"nil": {
// 			expErr: errors.New("nil"),
// 		},
// 		"incompatible versions": {
// 			api: &mockAPI{
// 				runtimeVer: 0x010000,
// 				compVer:    0x020000,
// 			},
// 			expErr: errors.New("incompatible"),
// 		},
// 		"different but compatible versions": {
// 			api: &mockAPI{
// 				runtimeVer:     0x0E000,
// 				compVer:        0x0F000,
// 				newTopo:        &mockTopology{},
// 				newTopoCleanup: testCleanup,
// 			},
// 			expTimesLocked: 1,
// 			expTopology:    &mockTopology{},
// 		},
// 		"newTopology failed": {
// 			api: &mockAPI{
// 				newTopoErr: errors.New("mock newTopology"),
// 			},
// 			expTimesLocked: 1,
// 			expErr:         errors.New("mock newTopology"),
// 		},
// 		"SetFlags failed": {
// 			api: &mockAPI{
// 				newTopo: &mockTopology{
// 					setFlagsErr: errors.New("mock SetFlags"),
// 				},
// 				newTopoCleanup: testCleanup,
// 			},
// 			expTimesCleanupCalled: 1,
// 			expTimesLocked:        1,
// 			expErr:                errors.New("mock SetFlags"),
// 		},
// 		"Load failed": {
// 			api: &mockAPI{
// 				newTopo: &mockTopology{
// 					loadErr: errors.New("mock Load"),
// 				},
// 				newTopoCleanup: testCleanup,
// 			},
// 			expTimesCleanupCalled: 1,
// 			expTimesLocked:        1,
// 			expErr:                errors.New("mock Load"),
// 		},
// 		"success": {
// 			api: &mockAPI{
// 				newTopo:        &mockTopology{},
// 				newTopoCleanup: testCleanup,
// 			},
// 			expTimesLocked: 1,
// 			expTopology:    &mockTopology{},
// 		},
// 	} {
// 		t.Run(name, func(t *testing.T) {
// 			testCleanupCalled = 0

// 			topo, cleanup, err := GetTopology(tc.api)

// 			common.CmpErr(t, tc.expErr, err)
// 			if diff := cmp.Diff(topo, tc.expTopology, cmp.AllowUnexported(mockTopology{})); diff != "" {
// 				t.Fatalf("(-want, +got): %v\n", diff)
// 			}

// 			if tc.expTopology == nil {
// 				common.AssertTrue(t, cleanup == nil, "")
// 			} else {
// 				common.AssertTrue(t, cleanup != nil, "")
// 			}

// 			common.AssertEqual(t, tc.expTimesCleanupCalled, testCleanupCalled, "")

// 			if mockAPI, ok := tc.api.(*mockAPI); ok {
// 				common.AssertEqual(t, tc.expTimesLocked, mockAPI.numCallsLock, "")
// 				common.AssertEqual(t, tc.expTimesLocked, mockAPI.numCallsUnlock, "")
// 			}
// 		})
// 	}
// }
