//
// (C) Copyright 2021 Intel Corporation.
//
// SPDX-License-Identifier: BSD-2-Clause-Patent
//

package hwloc

/*
#cgo LDFLAGS: -lhwloc
#include <hwloc.h>

int cmpt_get_osdev_type(hwloc_obj_t node)
{
	if (node->attr == NULL) {
		return -1;
	}

	return node->attr->osdev.type;
}

struct hwloc_pcidev_attr_s *cmpt_get_pcidev_attr(hwloc_obj_t node) {
	if (node->attr == NULL) {
		return NULL;
	}

	return &node->attr->pcidev;
}

#if HWLOC_API_VERSION >= 0x00020000

int cmpt_setFlags(hwloc_topology_t topology)
{
	return hwloc_topology_set_all_types_filter(topology, HWLOC_TYPE_FILTER_KEEP_ALL);
}

int cmpt_get_arity(hwloc_obj_t node)
{
	return node->io_arity;
}

int cmpt_get_parent_arity(hwloc_obj_t node)
{
	return node->parent->io_arity;
}

hwloc_obj_t cmpt_get_sibling(hwloc_obj_t node, int idx)
{
	hwloc_obj_t child;
	int i;

	child = node->parent->io_first_child;
	for (i = 0; i < idx; i++) {
		child = child->next_sibling;
	}
	return child;
}

hwloc_obj_t cmpt_get_child(hwloc_obj_t node, int idx)
{
	hwloc_obj_t child;
	int i;

	child = node->io_first_child;
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

int cmpt_get_arity(hwloc_obj_t node)
{
	return node->arity;
}

int cmpt_get_parent_arity(hwloc_obj_t node)
{
	return node->parent->arity;
}

hwloc_obj_t cmpt_get_sibling(hwloc_obj_t node, int idx)
{
	return node->parent->children[idx];
}

hwloc_obj_t cmpt_get_child(hwloc_obj_t node, int idx)
{
	return node->children[idx];
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

func (t *topology) getProcessCPUSet(pid int32, flags int) (*cpuSet, func(), error) {
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

func (t *topology) getNextObjByType(objType int, prev *object) (*object, error) {
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

func (t *topology) getNumObjByType(objType int) uint {
	return uint(C.hwloc_get_nbobjs_by_type(t.cTopology, C.hwloc_obj_type_t(objType)))
}

const (
	objTypeOSDevice  = C.HWLOC_OBJ_OS_DEVICE
	objTypePCIDevice = C.HWLOC_OBJ_PCI_DEVICE
	objTypeNUMANode  = C.HWLOC_OBJ_NUMANODE
	objTypeCore      = C.HWLOC_OBJ_CORE

	osDevTypeNetwork     = C.HWLOC_OBJ_OSDEV_NETWORK
	osDevTypeOpenFabrics = C.HWLOC_OBJ_OSDEV_OPENFABRICS
)

type rawTopology interface {
	raw() C.hwloc_topology_t
}

// object is a thin wrapper for hwloc_obj_t and related functions.
type object struct {
	cObj C.hwloc_obj_t
	topo rawTopology
}

func (o *object) name() string {
	return C.GoString(o.cObj.name)
}

func (o *object) osIndex() uint {
	return uint(o.cObj.os_index)
}

func (o *object) logicalIndex() uint {
	return uint(o.cObj.logical_index)
}

func (o *object) getNumSiblings() uint {
	return uint(C.cmpt_get_parent_arity(o.cObj))
}

func (o *object) getSibling(index uint) (*object, error) {
	cResult := C.cmpt_get_sibling(o.cObj, C.int(index))
	if cResult == nil {
		return nil, errors.Errorf("sibling of object %q not found", o.name())
	}

	return newObject(o.topo, cResult), nil
}

func (o *object) getNumChildren() uint {
	return uint(C.cmpt_get_arity(o.cObj))
}

func (o *object) getChild(index uint) (*object, error) {
	if o.cObj.children == nil {
		return nil, errors.Errorf("object %q has no children", o.name())
	}

	if index >= o.getNumChildren() {
		return nil, errors.Errorf("index %d not possible; object %q has %d children", index, o.name(), o.getNumChildren())
	}

	cResult := C.cmpt_get_child(o.cObj, C.int(index))
	return newObject(o.topo, cResult), nil
}

func (o *object) cpuSet() *cpuSet {
	return newCPUSet(o.topo, o.cObj.cpuset)
}

func (o *object) nodeSet() *nodeSet {
	return newNodeSet(o.cObj.nodeset)
}

func (o *object) objType() int {
	return int(o.cObj._type)
}

func (o *object) osDevType() (int, error) {
	if o.objType() != objTypeOSDevice {
		return 0, errors.Errorf("device %q is not an OS Device", o.name())
	}
	devType := C.cmpt_get_osdev_type(o.cObj)
	if devType < 0 {
		return 0, errors.Errorf("device %q attrs are nil", o.name())
	}
	return int(devType), nil
}

func (o *object) pciAddr() (string, error) {
	if o.objType() != objTypePCIDevice {
		return "", errors.Errorf("device %q is not a PCI Device", o.name())
	}
	pciDevAttr := C.cmpt_get_pcidev_attr(o.cObj)
	if pciDevAttr == nil {
		return "", errors.Errorf("device %q attrs are nil", o.name())
	}
	return fmt.Sprintf("%04d:%02d:%02d.%02d", pciDevAttr.domain, pciDevAttr.bus,
		pciDevAttr.dev, pciDevAttr._func), nil
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
