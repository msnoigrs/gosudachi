package dartsclone

const (
	unitLength = 32
)

type bitVector struct {
	units   []uint32
	ranks   []int
	numOnes int
	length  int
}

func newBitVector() *bitVector {
	return &bitVector{}
}

func (v *bitVector) get(id int) bool {
	return v.units[id/unitLength]>>((uint(id)%unitLength)&1) == 1
}

func (v *bitVector) rank(id int) int {
	const mask = uint32(0xffffffff)
	unitId := id / unitLength
	offset := uint(id % unitLength)
	return v.ranks[unitId] + popCount(v.units[unitId] & ^(mask<<offset))
}

func (v *bitVector) set(id int, bit bool) {
	if bit {
		v.units[id/unitLength] |= uint32(1) << uint(id%unitLength)
	} else {
		v.units[id/unitLength] &= ^(uint32(1) << uint(id%unitLength))
	}
}

func (v *bitVector) extend() {
	if (v.length % unitLength) == 0 {
		v.units = append(v.units, 0)
	}
	v.length++
}

func (v *bitVector) build() {
	v.ranks = make([]int, len(v.units), len(v.units))
	v.numOnes = 0
	for i := 0; i < len(v.units); i++ {
		v.ranks[i] = v.numOnes
		v.numOnes += popCount(v.units[i])
	}
}

func (v *bitVector) clear() {
	v.units = v.units[:0]
	v.ranks = []int{}
}

func popCount(unit uint32) int {
	unit = ((unit & 0xAAAAAAAA) >> 1) + (unit & 0x55555555)
	unit = ((unit & 0xCCCCCCCC) >> 2) + (unit & 0x33333333)
	unit = ((unit >> 4) + unit) & 0x0F0F0F0F
	unit += unit >> 8
	unit += unit >> 16
	return int(unit & 0xFF)
}
