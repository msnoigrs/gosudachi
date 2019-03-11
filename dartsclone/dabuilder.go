package dartsclone

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	blockSize      = 256
	numExtraBlocks = 16
	numExtras      = blockSize * numExtraBlocks
	upperMask      = 0xFF << 21
	lowerMask      = 0xFF
)

type ProgressFunc func(state int, max int)

func dabuSetHasLeaf(u uint32, hasLeaf bool) uint32 {
	if hasLeaf {
		return u | uint32(1)<<8
	}
	return u & ^(uint32(1) << 8)
}

func dabuSetValue(value int) uint32 {
	return uint32(value) | (1 << 31)
}

func dabuSetLabel(u uint32, label byte) uint32 {
	return u & ^uint32(0xFF) | uint32(label)
}

func dabuSetOffset(u uint32, offset int) uint32 {
	u &= (uint32(1) << 31) | (uint32(1) << 8) | uint32(0xFF)
	if uint32(offset) < (uint32(1) << 21) {
		return u | uint32(offset)<<10
	}
	return u | (uint32(offset) << 2) | (uint32(1) << 9)
}

type keySet struct {
	keys   [][]byte
	values []int
}

func newKeySet(keys [][]byte, values []int) *keySet {
	return &keySet{
		keys:   keys,
		values: values,
	}
}

func (ks *keySet) length() int {
	return len(ks.keys)
}

func (ks *keySet) getKey(id int) []byte {
	return ks.keys[id]
}

func (ks *keySet) getKeyByte(keyId int, byteId int) byte {
	if byteId >= len(ks.keys[keyId]) {
		return 0
	}
	return ks.keys[keyId][byteId]
}

func (ks *keySet) hasValues() bool {
	return len(ks.values) > 0
}

func (ks *keySet) getValue(id int) int {
	if len(ks.values) > 0 {
		return ks.values[id]
	}
	return id
}

type extraUnit struct {
	prev    int
	next    int
	isFixed bool
	isUsed  bool
}

type doubleArrayBuilder struct {
	progressFunction ProgressFunc
	units            []uint32
	extras           []extraUnit
	labels           []byte
	table            []int
	extrasHead       int
}

func newDoubleArrayBuilder(f ProgressFunc) *doubleArrayBuilder {
	return &doubleArrayBuilder{
		progressFunction: f,
	}
}

func (dab *doubleArrayBuilder) build(ks *keySet) ([]uint32, error) {
	if ks.hasValues() {
		dawgBuilder := newDAWGBuilder()
		err := dab.buildDAWG(ks, dawgBuilder)
		if err != nil {
			return []uint32{}, err
		}
		dab.buildFromDAWGHeader(dawgBuilder)
		dawgBuilder.clear()
	} else {
		err := dab.buildFromKeySetHeader(ks)
		if err != nil {
			return []uint32{}, err
		}
	}
	return dab.units, nil
}

func (dab *doubleArrayBuilder) copyBuffer() []byte {
	buffer := bytes.NewBuffer(make([]byte, 0, len(dab.units)*4))
	for _, u := range dab.units {
		_ = binary.Write(buffer, binary.LittleEndian, u)
	}
	return buffer.Bytes()
}

func (dab *doubleArrayBuilder) clear() {
	dab.units = []uint32{}
	dab.extras = []extraUnit{}
	dab.labels = []byte{}
	dab.table = []int{}
}

func (dab *doubleArrayBuilder) numBlocks() int {
	return len(dab.units) / blockSize
}

func (dab *doubleArrayBuilder) getExtras(id int) *extraUnit {
	return &dab.extras[id%numExtras]
}

func (dab *doubleArrayBuilder) buildDAWG(ks *keySet, db *dawgBuilder) error {
	db.initialize()
	var max int
	if dab.progressFunction != nil {
		max = ks.length() + 1
	}
	for i := 0; i < ks.length(); i++ {
		err := db.insert(ks.getKey(i), ks.getValue(i))
		if err != nil {
			return err
		}
		if dab.progressFunction != nil {
			dab.progressFunction(i+1, max)
		}
	}
	db.finish()
	return nil
}

func (dab *doubleArrayBuilder) buildFromDAWGHeader(db *dawgBuilder) {
	numUnits := 1
	for numUnits < db.length() {
		numUnits *= 2
	}
	dab.units = make([]uint32, 0, numUnits)

	dab.table = make([]int, db.numIntersections())

	dab.extras = make([]extraUnit, numExtras)

	dab.reserveId(0)
	dab.getExtras(0).isUsed = true
	dab.units[0] = dabuSetOffset(dab.units[0], 1)
	dab.units[0] = dabuSetLabel(dab.units[0], 0)

	if db.child(dawgRoot) != 0 {
		dab.buildFromDAWGInsert(db, dawgRoot, 0)
	}

	dab.fixAllBlocks()

	dab.extras = []extraUnit{}
	dab.labels = []byte{}
	dab.table = []int{}
}

func (dab *doubleArrayBuilder) buildFromDAWGInsert(db *dawgBuilder, dawgId int, dicId int) {
	dawgChildId := int(db.child(dawgId))
	if db.isIntersection(dawgChildId) {
		intersectionId := db.intersectionId(dawgChildId)
		offset := dab.table[intersectionId]
		if offset != 0 {
			offset ^= dicId
			if (offset&upperMask) == 0 || (offset&lowerMask) == 0 {
				if db.isLeaf(dawgChildId) {
					dab.units[dicId] = dabuSetHasLeaf(dab.units[dicId], true)
				}
				dab.units[dicId] = dabuSetOffset(dab.units[dicId], offset)
				return
			}
		}
	}

	offset := dab.arrangeFromDAWG(db, dawgId, dicId)
	if db.isIntersection(dawgChildId) {
		dab.table[db.intersectionId(dawgChildId)] = offset
	}

	for {
		childLabel := db.label(dawgChildId)
		dicChildId := offset ^ int(childLabel)
		if childLabel != 0 {
			dab.buildFromDAWGInsert(db, dawgChildId, dicChildId)
		}
		dawgChildId = db.sibling(dawgChildId)
		if dawgChildId == 0 {
			break
		}
	}
}

func (dab *doubleArrayBuilder) arrangeFromDAWG(db *dawgBuilder, dawgId int, dicId int) int {
	for i := 0; i < len(dab.labels); i++ {
		dab.labels[i] = 0
	}
	dab.labels = dab.labels[:0]

	dawgChildId := int(db.child(dawgId))
	for dawgChildId != 0 {
		dab.labels = append(dab.labels, db.label(dawgChildId))
		dawgChildId = db.sibling(int(dawgChildId))
	}

	offset := dab.findValidOffset(dicId)
	dab.units[dicId] = dabuSetOffset(dab.units[dicId], dicId^offset)

	dawgChildId = int(db.child(dawgId))
	for _, label := range dab.labels {
		dicChildId := offset ^ int(label)
		dab.reserveId(dicChildId)

		if db.isLeaf(dawgChildId) {
			dab.units[dicId] = dabuSetHasLeaf(dab.units[dicId], true)
			dab.units[dicChildId] = dabuSetValue(int(db.value(dawgChildId)))
		} else {
			dab.units[dicChildId] = dabuSetLabel(dab.units[dicChildId], label)
		}

		dawgChildId = db.sibling(dawgChildId)
	}
	dab.getExtras(offset).isUsed = true

	return offset
}

func (dab *doubleArrayBuilder) buildFromKeySetHeader(ks *keySet) error {
	numUnits := 1
	for numUnits < ks.length() {
		numUnits *= 2
	}
	dab.units = make([]uint32, 0, numUnits)

	dab.extras = make([]extraUnit, numExtras)

	dab.reserveId(0)
	dab.getExtras(0).isUsed = true
	dab.units[0] = dabuSetOffset(dab.units[0], 1)
	dab.units[0] = dabuSetLabel(dab.units[0], 0)

	if ks.length() > 0 {
		err := dab.buildFromKeySetInsert(ks, 0, ks.length(), 0, 0)
		if err != nil {
			return err
		}
	}

	dab.fixAllBlocks()

	dab.extras = []extraUnit{}
	dab.labels = []byte{}

	return nil
}

func (dab *doubleArrayBuilder) buildFromKeySetInsert(ks *keySet, begin int, end int, depth int, dicId int) error {
	offset, err := dab.arrangeFromKeySet(ks, begin, end, depth, dicId)
	if err != nil {
		return err
	}

	for begin < end {
		if ks.getKeyByte(begin, depth) != 0 {
			break
		}
		begin++
	}
	if begin == end {
		return nil
	}

	lastBegin := begin
	lastLabel := ks.getKeyByte(begin, depth)
	for begin++; begin < end; begin++ {
		label := ks.getKeyByte(begin, depth)
		if label != lastLabel {
			err = dab.buildFromKeySetInsert(ks, lastBegin, begin, depth+1, offset^int(lastLabel))
			if err != nil {
				return err
			}
			lastBegin = begin
			lastLabel = ks.getKeyByte(begin, depth)
		}
	}
	return dab.buildFromKeySetInsert(ks, lastBegin, end, depth+1, offset^int(lastLabel))
}

func (dab *doubleArrayBuilder) arrangeFromKeySet(ks *keySet, begin int, end int, depth int, dicId int) (int, error) {
	for i := 0; i < len(dab.labels); i++ {
		dab.labels[i] = 0
	}
	dab.labels = dab.labels[:0]

	value := -1
	for i := begin; i < end; i++ {
		label := ks.getKeyByte(i, depth)
		if label == 0 {
			if depth < len(ks.getKey(i)) {
				return 0, fmt.Errorf("invalid null character")
			} else if ks.getValue(i) < 0 {
				return 0, fmt.Errorf("negative value")
			}

			if value == -1 {
				value = ks.getValue(i)
			}
			if dab.progressFunction != nil {
				dab.progressFunction(i+1, ks.length()+1)
			}
		}

		if len(dab.labels) == 0 {
			dab.labels = append(dab.labels, label)
		} else if label != dab.labels[len(dab.labels)-1] {
			if label < dab.labels[len(dab.labels)-1] {
				return 0, fmt.Errorf("wrong key order")
			}
			dab.labels = append(dab.labels, label)
		}
	}

	offset := dab.findValidOffset(dicId)
	dab.units[dicId] = dabuSetOffset(dab.units[dicId], dicId^offset)

	for _, label := range dab.labels {
		dicChildId := offset ^ int(label)
		dab.reserveId(dicChildId)

		if label == 0 {
			dab.units[dicId] = dabuSetHasLeaf(dab.units[dicId], true)
			dab.units[dicChildId] = dabuSetValue(value)
		} else {
			dab.units[dicChildId] = dabuSetLabel(dab.units[dicChildId], label)
		}
	}
	dab.getExtras(offset).isUsed = true

	return offset, nil
}

func (dab *doubleArrayBuilder) findValidOffset(id int) int {
	if dab.extrasHead >= len(dab.units) {
		return len(dab.units) | (id & lowerMask)
	}

	unfixedId := dab.extrasHead
	for {
		offset := unfixedId ^ int(dab.labels[0])
		if dab.isValidOffset(id, offset) {
			return offset
		}
		unfixedId = dab.getExtras(unfixedId).next
		if unfixedId == dab.extrasHead {
			break
		}
	}

	return len(dab.units) | (id & lowerMask)
}

func (dab *doubleArrayBuilder) isValidOffset(id int, offset int) bool {
	if dab.getExtras(offset).isUsed {
		return false
	}

	relOffset := id ^ offset
	if (relOffset&lowerMask) != 0 && (relOffset&upperMask) != 0 {
		return false
	}

	for i := 1; i < len(dab.labels); i++ {
		if dab.getExtras(offset ^ int(dab.labels[i])).isFixed {
			return false
		}
	}
	return true
}

func (dab *doubleArrayBuilder) reserveId(id int) {
	if id >= len(dab.units) {
		dab.expandUnits()
	}

	if id == dab.extrasHead {
		dab.extrasHead = dab.getExtras(id).next
		if dab.extrasHead == id {
			dab.extrasHead = len(dab.units)
		}
	}
	dab.getExtras(dab.getExtras(id).prev).next = dab.getExtras(id).next
	dab.getExtras(dab.getExtras(id).next).prev = dab.getExtras(id).prev
	dab.getExtras(id).isFixed = true
}

func (dab *doubleArrayBuilder) expandUnits() {
	srcNumUnits := len(dab.units)
	srcNumBlocks := dab.numBlocks()

	destNumUnits := srcNumUnits + blockSize
	destNumBlocks := srcNumBlocks + 1

	if destNumBlocks > numExtraBlocks {
		dab.fixBlock(srcNumBlocks - numExtraBlocks)
	}

	dab.units = append(dab.units, make([]uint32, destNumUnits-srcNumUnits)...)
	if destNumBlocks > numExtraBlocks {
		for id := srcNumUnits; id < destNumUnits; id++ {
			e := dab.getExtras(id)
			e.isUsed = false
			e.isFixed = false
		}
	}

	for i := srcNumUnits + 1; i < destNumUnits; i++ {
		dab.getExtras(i - 1).next = i
		dab.getExtras(i).prev = i - 1
	}

	dab.getExtras(srcNumUnits).prev = destNumUnits - 1 // XXX: ???
	dab.getExtras(destNumUnits - 1).next = srcNumUnits

	dab.getExtras(srcNumUnits).prev = dab.getExtras(dab.extrasHead).prev
	dab.getExtras(destNumUnits - 1).next = dab.extrasHead

	dab.getExtras(dab.getExtras(dab.extrasHead).prev).next = srcNumUnits
	dab.getExtras(dab.extrasHead).prev = destNumUnits - 1
}

func (dab *doubleArrayBuilder) fixAllBlocks() {
	begin := 0
	end := dab.numBlocks()
	if end > numExtraBlocks {
		begin = end - numExtraBlocks
	}

	for blockId := begin; blockId != end; blockId++ {
		dab.fixBlock(blockId)
	}
}

func (dab *doubleArrayBuilder) fixBlock(blockId int) {
	begin := blockId * blockSize
	end := begin + blockSize

	var unusedOffset int
	for offset := begin; offset != end; offset++ {
		if !dab.getExtras(offset).isUsed {
			unusedOffset = offset
			break
		}
	}

	for id := begin; id != end; id++ {
		if !dab.getExtras(id).isFixed {
			dab.reserveId(id)
			dab.units[id] = dabuSetLabel(dab.units[id], (byte(id ^ unusedOffset)))
		}
	}
}
