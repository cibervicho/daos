//
// (C) Copyright 2021 Intel Corporation.
//
// SPDX-License-Identifier: BSD-2-Clause-Patent
//

package hwloc

/*
#cgo LDFLAGS: -lhwloc
#include <hwloc.h>

#if HWLOC_API_VERSION >= 0x00020000

int cmpt_setFlags(hwloc_topology_t topology)
{
	return hwloc_topology_set_all_types_filter(topology, HWLOC_TYPE_FILTER_KEEP_ALL);
}

hwloc_obj_t cmpt_get_obj_by_depth(hwloc_topology_t topology, int depth, uint idx)
{
	return hwloc_get_obj_by_depth(topology, depth, idx);
}

uint cmpt_get_nbobjs_by_depth(hwloc_topology_t topology, int depth)
{
	return (uint)hwloc_get_nbobjs_by_depth(topology, depth);
}

int cmpt_get_parent_arity(hwloc_obj_t node)
{
	return node->parent->io_arity;
}

hwloc_obj_t cmpt_get_child(hwloc_obj_t node, int idx)
{
	hwloc_obj_t child;
	int i;

	child = node->parent->io_first_child;
	for (i = 0; i < idx; i++) {
		child = child->next_sibling;
	}
	return child;
}
#else
int cmpt_setFlags(hwloc_topology_t topology)
{
	return hwloc_topology_set_flags(topology, HWLOC_TOPOLOGY_FLAG_IO_DEVICES);
}

hwloc_obj_t cmpt_get_obj_by_depth(hwloc_topology_t topology, int depth, uint idx)
{
	return hwloc_get_obj_by_depth(topology, (uint)depth, idx);
}

uint cmpt_get_nbobjs_by_depth(hwloc_topology_t topology, int depth)
{
	return (uint)hwloc_get_nbobjs_by_depth(topology, (uint)depth);
}

int cmpt_get_parent_arity(hwloc_obj_t node)
{
	return node->parent->arity;
}

hwloc_obj_t cmpt_get_child(hwloc_obj_t node, int idx)
{
	return node->parent->children[idx];
}
#endif
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/pkg/errors"
)

type api struct {
	sync.Mutex
}

func (a *api) runtimeVersion() uint {
	return uint(C.hwloc_get_api_version())
}

func (a *api) compiledVersion() uint {
	return C.HWLOC_API_VERSION
}

func (a *api) newTopology() (*topology, func(), error) {
	topo := &topology{
		api: a,
	}
	status := C.hwloc_topology_init(&topo.cTopology)
	if status != 0 {
		return nil, nil, errors.Errorf("hwloc topology init failed: %v", status)
	}

	return topo, func() {
		C.hwloc_topology_destroy(topo.cTopology)
	}, nil
}

// topology is a thin wrapper for the hwloc topology and related functions.
type topology struct {
	api       *api
	cTopology C.hwloc_topology_t
}

func (t *topology) raw() C.hwloc_topology_t {
	return t.cTopology
}

func (t *topology) GetProcessCPUSet(pid int32, flags int) (*cpuSet, func(), error) {
	// Access to hwloc_get_proc_cpubind must be synchronized
	t.api.Lock()
	defer t.api.Unlock()

	cpuset := C.hwloc_bitmap_alloc()
	if cpuset == nil {
		return nil, nil, errors.New("hwloc_bitmap_alloc failed")
	}
	cleanup := func() {
		C.hwloc_bitmap_free(cpuset)
	}

	if status := C.hwloc_get_proc_cpubind(t.cTopology, C.int(pid), cpuset, C.int(flags)); status != 0 {
		cleanup()
		return nil, nil, errors.Errorf("hwloc get proc cpubind failed: %v", status)
	}

	return newCPUSet(t, cpuset), cleanup, nil
}

func (t *topology) setFlags() error {
	if status := C.cmpt_setFlags(t.cTopology); status != 0 {
		return errors.Errorf("hwloc set flags failed: %v", status)
	}
	return nil
}

func (t *topology) load() error {
	if status := C.hwloc_topology_load(t.cTopology); status != 0 {
		return errors.Errorf("hwloc topology load failed: %v", status)
	}
	return nil
}

func (t *topology) GetObjByDepth(depth int, index uint) (*object, error) {
	obj := C.cmpt_get_obj_by_depth(t.cTopology, C.int(depth), C.uint(index))
	if obj == nil {
		return nil, errors.Errorf("no hwloc object found with depth=%d, index=%d", depth, index)
	}
	return newObject(t, obj), nil
}

func (t *topology) GetNextObjByType(objType ObjType, prev *object) (*object, error) {
	var cPrev C.hwloc_obj_t
	if prev != nil {
		cPrev = prev.cObj
	}
	obj := C.hwloc_get_next_obj_by_type(t.cTopology, C.hwloc_obj_type_t(objType), cPrev)
	if obj == nil {
		return nil, errors.Errorf("no next hwloc object found with type=%d", objType)
	}
	return newObject(t, obj), nil
}

func (t *topology) GetTypeDepth(objType ObjType) int {
	return int(C.hwloc_get_type_depth(t.cTopology, C.hwloc_obj_type_t(objType)))
}

func (t *topology) GetNumObjAtDepth(depth int) uint {
	return uint(C.cmpt_get_nbobjs_by_depth(t.cTopology, C.int(depth)))
}

func (t *topology) getNumObjByType(objType ObjType) uint {
	return uint(C.hwloc_get_nbobjs_by_type(t.cTopology, C.hwloc_obj_type_t(objType)))
}

type ObjType uint

const (
	ObjTypeOSDevice  = ObjType(C.HWLOC_OBJ_OS_DEVICE)
	ObjTypePCIDevice = ObjType(C.HWLOC_OBJ_PCI_DEVICE)
	ObjTypeNUMANode  = ObjType(C.HWLOC_OBJ_NUMANODE)
	ObjTypeCore      = ObjType(C.HWLOC_OBJ_CORE)

	TypeDepthOSDevice = C.HWLOC_TYPE_DEPTH_OS_DEVICE
	TypeDepthUnknown  = C.HWLOC_TYPE_DEPTH_UNKNOWN

	OSDevTypeNetwork    = C.HWLOC_OBJ_OSDEV_NETWORK
	OSDevTypeOpenFabric = C.HWLOC_OBJ_OSDEV_OPENFABRIC
)

type rawTopology interface {
	raw() C.hwloc_topology_t
}

// object is a thin wrapper for hwloc_obj_t and related functions.
type object struct {
	cObj C.hwloc_obj_t
	topo rawTopology
}

func (o *object) Name() string {
	return C.GoString(o.cObj.name)
}

func (o *object) OSIndex() uint {
	return uint(o.cObj.os_index)
}

func (o *object) LogicalIndex() uint {
	return uint(o.cObj.logical_index)
}

func (o *object) GetNumSiblings() uint {
	return uint(C.cmpt_get_parent_arity(o.cObj))
}

func (o *object) GetChild(index uint) (*object, error) {
	cResult := C.cmpt_get_child(o.cObj, C.int(index))
	if cResult == nil {
		return nil, errors.Errorf("child of object %q not found", o.Name())
	}

	return newObject(o.topo, cResult), nil
}

func (o *object) GetAncestorByType(objType ObjType) (*object, error) {
	cResult := C.hwloc_get_ancestor_obj_by_type(o.topo.raw(), C.hwloc_obj_type_t(objType), o.cObj)
	if cResult == nil {
		return nil, errors.Errorf("type %v ancestor of object %q not found", objType, o.Name())
	}

	return newObject(o.topo, cResult), nil
}

func (o *object) GetNonIOAncestor() (*object, error) {
	ancestorNode := C.hwloc_get_non_io_ancestor_obj(o.topo.raw(), o.cObj)
	if ancestorNode == nil {
		return nil, errors.New("unable to find non-io ancestor node for device")
	}

	return newObject(o.topo, ancestorNode), nil
}

func (o *object) CPUSet() *cpuSet {
	return newCPUSet(o.topo, o.cObj.cpuset)
}

func (o *object) NodeSet() *nodeSet {
	return newNodeSet(o.cObj.nodeset)
}

func (o *object) Type() ObjType {
	return ObjType(o.cObj._type)
}

func (o *object) OSDevType() (int, error) {
	if o.Type() != ObjTypeOSDevice {
		return 0, errors.Errorf("device %q is not an OS Device", o.Name())
	}
	if o.cObj.attr == nil {
		return 0, errors.Errorf("device %q attrs are nil", o.Name())
	}
	return o.cObj.attr.osdev._type, nil
}

func (o *object) PCIAddr() (string, error) {
	if o.Type() != ObjTypePCIDevice {
		return "", errors.Errorf("device %q is not a PCI Device", o.Name())
	}
	if o.cObj.attr == nil {
		return "", errors.Errorf("device %q attrs are nil", o.Name())
	}
	return fmt.Sprintf("%04d:%02d:%02d.%02d", cObj.attr.pcidev.domain, cObj.attr.pcidev.bus,
		cObj.attr.pcidev.dev, cObj.attr.pcidev._func), nil
}

func newObject(topo rawTopology, cObj C.hwloc_obj_t) *object {
	if cObj == nil {
		panic("nil hwloc_obj_t")
	}
	return &object{
		cObj: cObj,
		topo: topo,
	}
}

type bitmap struct {
	cSet C.hwloc_bitmap_t
}

func (b *bitmap) raw() C.hwloc_bitmap_t {
	return b.cSet
}

func (b *bitmap) String() string {
	var str *C.char

	strLen := C.hwloc_bitmap_asprintf(&str, b.raw())
	if strLen <= 0 {
		return ""
	}
	defer C.free(unsafe.Pointer(str))
	return C.GoString(str)
}

func (b *bitmap) intersects(other *bitmap) bool {
	return C.hwloc_bitmap_intersects(b.raw(), other.raw()) != 0
}

func (b *bitmap) isSubsetOf(other *bitmap) bool {
	return C.hwloc_bitmap_isincluded(b.raw(), other.raw()) != 0
}

type cpuSet struct {
	bitmap
	topo rawTopology
}

func (c *cpuSet) Intersects(other *cpuSet) bool {
	return c.intersects(&other.bitmap)
}

func (c *cpuSet) IsSubsetOf(other *cpuSet) bool {
	return c.isSubsetOf(&other.bitmap)
}

func (c *cpuSet) ToNodeSet() (*nodeSet, func(), error) {
	nodeset := C.hwloc_bitmap_alloc()
	if nodeset == nil {
		return nil, nil, errors.New("hwloc_bitmap_alloc failed")
	}
	cleanup := func() {
		C.hwloc_bitmap_free(nodeset)
	}
	C.hwloc_cpuset_to_nodeset(c.topo.raw(), c.cSet, nodeset)

	return newNodeSet(nodeset), cleanup, nil
}

type nodeSet struct {
	bitmap
}

func (n *nodeSet) Intersects(other *nodeSet) bool {
	return n.intersects(&other.bitmap)
}

func newCPUSet(topo rawTopology, cSet C.hwloc_bitmap_t) *cpuSet {
	return &cpuSet{
		bitmap: bitmap{
			cSet: cSet,
		},
		topo: topo,
	}
}

func newNodeSet(cSet C.hwloc_bitmap_t) *nodeSet {
	return &nodeSet{
		bitmap: bitmap{
			cSet: cSet,
		},
	}
}
