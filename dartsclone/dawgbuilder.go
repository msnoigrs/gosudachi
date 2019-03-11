package dartsclone

import (
	"fmt"
)

const (
	initialTableSize = 1 << 10
	dawgRoot         = 0
)

type node struct {
	child      int
	sibling    int
	label      byte
	isState    bool
	hasSibling bool
}

func (n *node) reset() {
	n.child = 0
	n.sibling = 0
	n.label = 0
	n.isState = false
	n.hasSibling = false
}

func (n *node) unit() uint32 {
	var sibling uint32
	if n.hasSibling {
		sibling = 1
	}
	if n.label == 0 {
		return uint32(n.child)<<1 | sibling
	}
	var state uint32
	if n.isState {
		state = 2
	}
	return uint32(n.child)<<2 | state | sibling
}

type unit uint32

func (u unit) child() uint32 {
	return uint32(u) >> 2
}

func (u unit) hasSibling() bool {
	return (uint32(u) & 1) == 1
}

func (u unit) value() uint32 {
	return uint32(u) >> 1
}

func (u unit) isState() bool {
	return (uint32(u) & 2) == 2
}

type stack []int

func (s stack) top() int {
	return s[len(s)-1]
}

func (s stack) pop() stack {
	return s[:len(s)-1]
}

type dawgBuilder struct {
	nodes           []node
	units           []uint32
	labels          []byte
	isIntersections *bitVector
	table           []int
	nodeStack       stack
	recycleBin      stack
	numStates       int
}

func newDAWGBuilder() *dawgBuilder {
	return &dawgBuilder{
		isIntersections: newBitVector(),
		table:           make([]int, initialTableSize, initialTableSize),
	}
}

func (b *dawgBuilder) child(id int) uint32 {
	return unit(b.units[id]).child()
}

func (b *dawgBuilder) sibling(id int) int {
	if unit(b.units[id]).hasSibling() {
		return id + 1
	}
	return 0
}

func (b *dawgBuilder) value(id int) uint32 {
	return unit(b.units[id]).value()
}

func (b *dawgBuilder) isLeaf(id int) bool {
	return b.labels[id] == 0
}

func (b *dawgBuilder) label(id int) byte {
	return b.labels[id]
}

func (b *dawgBuilder) isIntersection(id int) bool {
	return b.isIntersections.get(id)
}

func (b *dawgBuilder) intersectionId(id int) int {
	return b.isIntersections.rank(id) - 1
}

func (b *dawgBuilder) numIntersections() int {
	return b.isIntersections.numOnes
}

func (b *dawgBuilder) length() int {
	return len(b.units)
}

func (b *dawgBuilder) initialize() {
	b.appendNode()
	b.appendUnit()

	b.numStates = 1

	b.nodes[0].label = 0xFF
	b.nodeStack = append(b.nodeStack, 0)
}

func (b *dawgBuilder) finish() {
	b.flush(0)

	b.units[0] = b.nodes[0].unit()
	b.labels[0] = b.nodes[0].label

	b.nodes = []node{}
	b.table = []int{}
	b.nodeStack = []int{}
	b.recycleBin = []int{}

	b.isIntersections.build()
}

func (b *dawgBuilder) insert(key []byte, value int) error {
	if value < 0 {
		return fmt.Errorf("negative value")
	}
	keylen := len(key)
	if keylen == 0 {
		return fmt.Errorf("zero-length key")
	}

	var id int
	var keyPos int

	for ; keyPos <= keylen; keyPos++ {
		childId := b.nodes[id].child
		if childId == 0 {
			break
		}

		var keyLabel byte
		if keyPos <= keylen {
			keyLabel = key[keyPos]
		}
		if keyPos < keylen && keyLabel == 0 {
			return fmt.Errorf("invalid null character")
		}

		unitLabel := b.nodes[childId].label
		if keyLabel < unitLabel {
			return fmt.Errorf("wrong key order")
		} else if keyLabel > unitLabel {
			b.nodes[childId].hasSibling = true
			b.flush(childId)
			break
		}
		id = childId
	}

	if keyPos > keylen {
		return nil
	}

	for ; keyPos <= keylen; keyPos++ {
		var keyLabel byte
		if keyPos < keylen {
			keyLabel = key[keyPos]
		}
		childId := b.appendNode()

		if b.nodes[id].child == 0 {
			b.nodes[childId].isState = true
		}
		b.nodes[childId].sibling = b.nodes[id].child
		b.nodes[childId].label = keyLabel
		b.nodes[id].child = childId
		b.nodeStack = append(b.nodeStack, childId)

		id = childId
	}
	b.nodes[id].child = value

	return nil
}

func (b *dawgBuilder) clear() {
	b.nodes = []node{}
	b.units = []uint32{}
	b.labels = []byte{}
	b.isIntersections = nil
	b.table = []int{}
	b.nodeStack = []int{}
	b.recycleBin = []int{}
}

func (b *dawgBuilder) flush(id int) {
	for {
		nodeId := b.nodeStack.top()
		if nodeId == id {
			break
		}
		b.nodeStack = b.nodeStack.pop()

		if b.numStates >= len(b.table)-len(b.table)/4 {
			b.expandTable()
		}

		var numSiblings int
		for i := nodeId; i != 0; i = b.nodes[i].sibling {
			numSiblings++
		}

		matchId, hashId := b.findNode(nodeId)

		if matchId != 0 {
			b.isIntersections.set(matchId, true)
		} else {
			var unitId int
			for i := 0; i < numSiblings; i++ {
				unitId = b.appendUnit()
			}
			for i := nodeId; i != 0; i = b.nodes[i].sibling {
				b.units[unitId] = b.nodes[i].unit()
				b.labels[unitId] = b.nodes[i].label
				unitId--
			}
			matchId = unitId + 1
			b.table[hashId] = matchId
			b.numStates++
		}

		var next int
		for i := nodeId; i != 0; i = next {
			next = b.nodes[i].sibling
			b.freeNode(i)
		}

		b.nodes[b.nodeStack.top()].child = matchId
	}
	b.nodeStack = b.nodeStack.pop()
}

func (b *dawgBuilder) expandTable() {
	tablesize := len(b.table) * 2
	b.table = make([]int, tablesize, tablesize)
	for id := 1; id < len(b.units); id++ {
		if b.labels[id] == 0 || unit(b.units[id]).isState() {
			hashId := b.findUnit(id)
			b.table[hashId] = id
		}
	}
}

func (b *dawgBuilder) findUnit(id int) int {
	hashId := b.hashUnit(id) % len(b.table)
	for ; ; hashId = (hashId + 1) % len(b.table) {
		unitId := b.table[hashId]
		if unitId == 0 {
			break
		}
	}
	return hashId
}

func (b *dawgBuilder) findNode(nodeId int) (int, int) {
	hashId := b.hashNode(nodeId) % len(b.table)
	for ; ; hashId = (hashId + 1) % len(b.table) {
		unitId := b.table[hashId]
		if unitId == 0 {
			break
		}

		if b.areEqual(nodeId, unitId) {
			return unitId, hashId
		}
	}
	return 0, hashId
}

func (b *dawgBuilder) areEqual(nodeId int, unitId int) bool {
	for i := b.nodes[nodeId].sibling; i != 0; i = b.nodes[i].sibling {
		if !unit(b.units[unitId]).hasSibling() {
			return false
		}
		unitId++
	}
	if unit(b.units[unitId]).hasSibling() {
		return false
	}

	for i := nodeId; i != 0; i = b.nodes[i].sibling {
		if b.nodes[i].unit() != b.units[unitId] ||
			b.nodes[i].label != b.labels[unitId] {
			return false
		}
		unitId--
	}
	return true
}

func (b *dawgBuilder) hashUnit(id int) int {
	var hashValue int
	for ; id != 0; id++ {
		u := b.units[id]
		label := b.labels[id]
		hashValue ^= hash((uint32(label) << 24) ^ u)

		if !unit(u).hasSibling() {
			break
		}
	}
	return hashValue
}

func (b *dawgBuilder) hashNode(id int) int {
	var hashValue int
	for ; id != 0; id = b.nodes[id].sibling {
		u := b.nodes[id].unit()
		label := b.nodes[id].label
		hashValue ^= hash((uint32(label) << 24) ^ u)
	}
	return hashValue
}

func (b *dawgBuilder) appendUnit() int {
	b.isIntersections.extend()
	b.units = append(b.units, 0)
	b.labels = append(b.labels, 0)
	return b.isIntersections.length - 1
}

func (b *dawgBuilder) appendNode() int {
	var id int
	if len(b.recycleBin) == 0 {
		id = len(b.nodes)
		b.nodes = append(b.nodes, node{})
	} else {
		id = b.recycleBin.top()
		b.nodes[id].reset()
		b.recycleBin = b.recycleBin.pop()
	}
	return id
}

func (b *dawgBuilder) freeNode(id int) {
	b.recycleBin = append(b.recycleBin, id)
}

func hash(key uint32) int {
	key = ^key + (key << 15)
	key = key ^ (key >> 12)
	key = key + (key << 2)
	key = key ^ (key >> 4)
	key = key * 2057
	key = key ^ (key >> 16)
	return int(key)
}
