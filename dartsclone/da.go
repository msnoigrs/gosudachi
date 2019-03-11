package dartsclone

import (
	"errors"
	"io"
	"os"

	// "math"
	"unsafe"

	"github.com/msnoigrs/gosudachi/internal/mmap"
)

type DoubleArray struct {
	array  []uint32
	buffer []byte
}

func NewDoubleArray() *DoubleArray {
	return &DoubleArray{}
}

func (da *DoubleArray) SetArray(array []uint32) {
	da.array = array
	da.buffer = asByteArray(array)
}

func (da *DoubleArray) SetBuffer(buffer []byte) {
	da.buffer = buffer
	da.array = asUInt32Array(buffer)
}

func (da *DoubleArray) Array() []uint32 {
	return da.array
}

func (da *DoubleArray) ByteArray() []byte {
	return da.buffer
}

func (da *DoubleArray) Clear() {
	da.buffer = []byte{}
	da.array = []uint32{}
}

func (da *DoubleArray) Length() int {
	return len(da.array)
}

func (da *DoubleArray) TotalSize() int {
	return len(da.buffer)
}

func (da *DoubleArray) Build(keys [][]byte, values []int, f ProgressFunc) error {
	var err error
	dab := newDoubleArrayBuilder(f)
	da.array, err = dab.build(newKeySet(keys, values))
	if err != nil {
		return err
	}
	da.buffer = asByteArray(da.array)

	return nil
}

func (da *DoubleArray) Open(f *os.File, position int64, totalSize int64) (err error) {
	if position < 0 {
		position = 0
	}
	if totalSize <= 0 {
		finfo, err := f.Stat()
		if err != nil {
			return err
		}
		totalSize = finfo.Size()
	}
	da.buffer, err = mmap.Mmap(f, false, position, totalSize)
	if err != nil {
		return err
	}
	// err = mmap.Madvise(da.buffer, false)
	// if err != nil {
	// 	return err
	// }
	da.array = asUInt32Array(da.buffer)

	return nil
}

func (da *DoubleArray) Close() error {
	err := mmap.Munmap(da.buffer)
	if err != nil {
		return err
	}
	da.buffer = []byte{}
	da.array = []uint32{}

	return nil
}

func (da *DoubleArray) Save(writer io.Writer) (int, error) {
	return writer.Write(da.buffer)
}

func (da *DoubleArray) ExactMatchSearch(key []byte) (int, int) {
	var nodePos uint32
	u := daunit(da.array[0])

	for _, k := range key {
		nodePos ^= u.offset() ^ uint32(k)
		u = daunit(da.array[int(nodePos)])
		if u.label() != k {
			return -1, 0
		}
	}
	if !u.hasLeaf() {
		return -1, 0
	}
	u = daunit(da.array[int(nodePos^u.offset())])
	return u.value(), len(key)
}

func (da *DoubleArray) CommonPrefixSearch(key []byte, offset int, maxNumResult int) [][2]int {
	result := make([][2]int, 0)

	var nodePos uint32
	u := daunit(da.array[0])
	nodePos ^= u.offset()
	for i := offset; i < len(key); i++ {
		k := key[i]
		nodePos ^= uint32(k)
		u = daunit(da.array[int(nodePos)])
		if u.label() != k {
			return result
		}

		nodePos ^= u.offset()
		if u.hasLeaf() && len(result) < maxNumResult {
			result = append(result, [2]int{daunit(da.array[int(nodePos)]).value(), i + 1})
		}
	}
	return result
}

func (da *DoubleArray) CommonPrefixSearchItr(key []byte, offset int) *Iterator {
	return newIterator(da.array, key, offset)
}

type Iterator struct {
	array   []uint32
	key     []byte
	offset  int
	nodePos uint32
	rvalue  int
	roffset int
	err     error
}

func newIterator(array []uint32, key []byte, offset int) *Iterator {
	var nodePos uint32
	u := daunit(array[0])
	nodePos ^= u.offset()
	return &Iterator{
		array:   array,
		key:     key,
		offset:  offset,
		nodePos: nodePos,
		rvalue:  -1,
	}
}

func (it *Iterator) Next() bool {
	if it.err != nil {
		return false
	}
	if it.rvalue == -1 {
		it.rvalue, it.roffset = it.getNext()
	}
	return it.rvalue != -1
}

func (it *Iterator) Get() (int, int) {
	var (
		rvalue  int
		roffset int
	)
	if it.rvalue == -1 {
		rvalue, roffset = it.getNext()
		if rvalue == -1 {
			it.err = errors.New("No more element")
			return rvalue, roffset
		}
	} else {
		rvalue = it.rvalue
		roffset = it.roffset
		it.rvalue = -1
		it.roffset = 0
	}
	return rvalue, roffset
}

func (it *Iterator) Err() error {
	return it.err
}

func (it *Iterator) getNext() (int, int) {
	for ; it.offset < len(it.key); it.offset++ {
		k := it.key[it.offset]
		it.nodePos ^= uint32(k)
		u := daunit(it.array[int(it.nodePos)])
		if u.label() != k {
			it.offset = len(it.key) // no more loop
			return -1, 0
		}

		it.nodePos ^= u.offset()
		if u.hasLeaf() {
			it.offset++
			rvalue := daunit(it.array[int(it.nodePos)]).value()
			roffset := it.offset
			return rvalue, roffset
		}
	}
	return -1, 0
}

func asUInt32Array(data []byte) []uint32 {
	var sl = struct {
		addr uintptr
		len  int
		cap  int
	}{uintptr(unsafe.Pointer(&data[0])), len(data) / 4, len(data) / 4}
	return *(*[]uint32)(unsafe.Pointer(&sl))
	// return (*[math.MaxUint32 / 4]uint32)(unsafe.Pointer(&data[0]))[:len(data) / 4]
}

func asByteArray(data []uint32) []byte {
	// Slice memory layout
	// Copied this snippet from golang/sys package
	var sl = struct {
		addr uintptr
		len  int
		cap  int
	}{uintptr(unsafe.Pointer(&data[0])), len(data) * 4, len(data) * 4}
	return *(*[]byte)(unsafe.Pointer(&sl))
	// return (*[math.MaxUint32]byte)(unsafe.Pointer(&data[0]))[:len(data) * 4]
}

type daunit uint32

func (u daunit) hasLeaf() bool {
	return ((uint32(u) >> 8) & uint32(1)) == 1
}

func (u daunit) value() int {
	return int(uint32(u) & ((uint32(1) << 31) - 1))
}

func (u daunit) label() byte {
	return byte(uint32(u) & 0xFF)
}

func (u daunit) offset() uint32 {
	return (uint32(u) >> 10) << ((uint32(u) & (uint32(1) << 9)) >> 6)
}
