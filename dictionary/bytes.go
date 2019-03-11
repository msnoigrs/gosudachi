package dictionary

import (
	"bytes"
	"encoding/binary"
	"unicode/utf16"
)

func bufferToInt16(bytebuffer []byte, offset int) (int, int16) {
	var ret int16
	offsetend := offset + 2
	_ = binary.Read(bytes.NewBuffer(bytebuffer[offset:offsetend]), binary.LittleEndian, &ret)
	return offsetend, ret
}

func bufferToUint16(bytebuffer []byte, offset int) (int, uint16) {
	var ret uint16
	offsetend := offset + 2
	_ = binary.Read(bytes.NewBuffer(bytebuffer[offset:offsetend]), binary.LittleEndian, &ret)
	return offsetend, ret
}

func bufferToInt32(bytebuffer []byte, offset int) (int, int32) {
	var ret int32
	offsetend := offset + 4
	_ = binary.Read(bytes.NewBuffer(bytebuffer[offset:offsetend]), binary.LittleEndian, &ret)
	return offsetend, ret
}

func bufferToUint32(bytebuffer []byte, offset int) (int, uint32) {
	var ret uint32
	offsetend := offset + 4
	_ = binary.Read(bytes.NewBuffer(bytebuffer[offset:offsetend]), binary.LittleEndian, &ret)
	return offsetend, ret
}

func bufferToInt64(bytebuffer []byte, offset int) (int, int64) {
	var ret int64
	offsetend := offset + 8
	_ = binary.Read(bytes.NewBuffer(bytebuffer[offset:offsetend]), binary.LittleEndian, &ret)
	return offsetend, ret
}

func bufferToUint64(bytebuffer []byte, offset int) (int, uint64) {
	var ret uint64
	offsetend := offset + 8
	_ = binary.Read(bytes.NewBuffer(bytebuffer[offset:offsetend]), binary.LittleEndian, &ret)
	return offsetend, ret
}

type bufferToStringFunc func(bytebuffer []byte, offset int) (int, string)

func bufferToString(bytebuffer []byte, offset int) (int, string) {
	// length := int(bytebuffer[offset])
	// offset++
	var length uint16
	offset, length = bufferToUint16(bytebuffer, offset)
	offsetend := offset + int(length)
	return offsetend, string(bytebuffer[offset:offsetend])
}

func bufferToStringUtf16(bytebuffer []byte, offset int) (int, string) {
	length := int(bytebuffer[offset])
	offset++

	// java compatible
	javainternal := make([]uint16, length, length)
	for i := 0; i < length; i++ {
		s := offset + 2*i
		_ = binary.Read(bytes.NewBuffer(bytebuffer[s:s+2]), binary.LittleEndian, &javainternal[i])
	}
	return offset + length*2, string(utf16.Decode(javainternal))
}

func bufferToInt32Array(bytebuffer []byte, offset int) (int, []int32) {
	length := int(bytebuffer[offset])
	offset++
	array := make([]int32, length, length)
	for i := 0; i < length; i++ {
		s := offset + 4*i
		_ = binary.Read(bytes.NewBuffer(bytebuffer[s:s+4]), binary.LittleEndian, &array[i])
	}
	return offset + 4*length, array
}
