package dictionary

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
)

const (
	posDepth            = 6
	InhibitedConnection = math.MaxInt16
)

var (
	BosParameter = []int16{0, 0, 0}
	EosParameter = []int16{0, 0, 0}
)

type Grammar struct {
	bytebuffer           []byte
	posList              [][]string
	connectTableBytes    []byte
	isCopiedConnectTable bool
	connectTableOffset   int
	leftIdSize           int16
	rightIdSize          int16
	CharCategory         *CharacterCategory
	StorageSize          int
}

func NewGrammar(bytebuffer []byte, offset int, utf16string bool) *Grammar {
	var bufferToStringF bufferToStringFunc
	if utf16string {
		bufferToStringF = bufferToStringUtf16
	} else {
		bufferToStringF = bufferToString
	}
	originalOffset := offset
	var posLen uint16
	offset, posLen = bufferToUint16(bytebuffer, offset)
	posLeni := int(posLen)
	posList := make([][]string, posLeni, posLeni)
	for i := 0; i < posLeni; i++ {
		pos := make([]string, posDepth, posDepth)
		for j := 0; j < posDepth; j++ {
			offset, pos[j] = bufferToStringF(bytebuffer, offset)
		}
		posList[i] = pos
	}
	var (
		leftIdSize  int16
		rightIdSize int16
	)
	offset, leftIdSize = bufferToInt16(bytebuffer, offset)
	offset, rightIdSize = bufferToInt16(bytebuffer, offset)

	return &Grammar{
		bytebuffer:           bytebuffer,
		posList:              posList,
		connectTableBytes:    bytebuffer,
		isCopiedConnectTable: false,
		connectTableOffset:   offset,
		leftIdSize:           leftIdSize,
		rightIdSize:          rightIdSize,
		StorageSize:          (offset - originalOffset) + 2*int(leftIdSize)*int(rightIdSize),
	}
}

func (g *Grammar) GetPartOfSpeechSize() int {
	return len(g.posList)
}

func (g *Grammar) GetPartOfSpeechString(posId int16) []string {
	return g.posList[posId]
}

func (g *Grammar) GetPartOfSpeechId(pos []string) int16 {
L:
	for i, p := range g.posList {
		for j := 0; j < posDepth; j++ {
			if p[j] != pos[j] {
				continue L
			}
		}
		return int16(i)
	}
	return int16(-1)
}

func (g *Grammar) GetPosId(posstrings ...string) int16 {
	return g.GetPartOfSpeechId(posstrings)
}

func (g *Grammar) GetConnectCost(leftId int16, rightId int16) int16 {
	s := g.connectTableOffset + int(leftId)*2 + 2*int(g.leftIdSize)*int(rightId)
	_, cost := bufferToInt16(g.connectTableBytes, s)
	return cost
}

func (g *Grammar) SetConnectCost(leftId int16, rightId int16, cost int16) {
	if !g.isCopiedConnectTable {
		g.copyConnectTable()
	}
	s := g.connectTableOffset + int(leftId)*2 + 2*int(g.leftIdSize)*int(rightId)
	binary.LittleEndian.PutUint16(g.connectTableBytes[s:], uint16(cost))
}

// syncronized ???
func (g *Grammar) copyConnectTable() {
	l := 2 * int(g.leftIdSize) * int(g.rightIdSize)
	newbuffer := make([]byte, l, l)
	s := g.connectTableOffset
	copy(newbuffer, g.connectTableBytes[s:s+l])
	g.connectTableBytes = newbuffer
	g.connectTableOffset = 0
	g.isCopiedConnectTable = true
}

func (g *Grammar) WritePOSTableTo(buffer *bytes.Buffer, utf16string bool) error {
	var writeStringF writeStringFunc
	if utf16string {
		writeStringF = writeStringUtf16
	} else {
		writeStringF = writeString
	}
	err := binary.Write(buffer, binary.LittleEndian, uint16(len(g.posList)))
	if err != nil {
		return err
	}

	for _, pos := range g.posList {
		for _, t := range pos {
			err := writeStringF(buffer, t)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *Grammar) WriteConnMatrixTo(writer io.Writer) (int, error) {
	err := binary.Write(writer, binary.LittleEndian, uint16(g.leftIdSize))
	if err != nil {
		return 0, err
	}
	err = binary.Write(writer, binary.LittleEndian, uint16(g.rightIdSize))
	if err != nil {
		return 2, err
	}
	n, err := writer.Write(g.connectTableBytes[g.connectTableOffset : g.connectTableOffset+2*int(g.leftIdSize)*int(g.rightIdSize)])
	if err != nil {
		return 4, err
	}
	return n + 4, nil
}
