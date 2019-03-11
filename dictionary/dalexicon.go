package dictionary

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/msnoigrs/gosudachi/dartsclone"
)

const (
	wordParameterListElementSize = 2 * 3
)

type wordIdTable struct {
	bytebuffer []byte
	size       int32
	//offset     int
}

func newWordIdTable(bytebuffer []byte, offset int) *wordIdTable {
	_, size := bufferToInt32(bytebuffer, offset)
	return &wordIdTable{
		bytebuffer: bytebuffer[offset+4 : offset+4+int(size)],
		size:       size,
		//offset:     offset + 4,
	}
}

func (t *wordIdTable) storageSize() int {
	return 4 + int(t.size)
}

func (t *wordIdTable) get(index int) []int32 {
	_, result := bufferToInt32Array(t.bytebuffer, index)
	return result
}

type wordParameterList struct {
	bytebuffer []byte
	size       int32
	offset     int
	isCopied   bool
}

func newWordParameterList(bytebuffer []byte, offset int) *wordParameterList {
	offset, size := bufferToInt32(bytebuffer, offset)
	return &wordParameterList{
		bytebuffer: bytebuffer,
		size:       size,
		offset:     offset,
		isCopied:   false,
	}
}

func (l *wordParameterList) storageSize() int {
	return 4 + wordParameterListElementSize*int(l.size)
}

func (l *wordParameterList) getLeftId(wordId int32) int16 {
	_, ret := bufferToInt16(l.bytebuffer, l.offset+wordParameterListElementSize*int(wordId))
	return ret
}

func (l *wordParameterList) getRightId(wordId int32) int16 {
	_, ret := bufferToInt16(l.bytebuffer, l.offset+wordParameterListElementSize*int(wordId)+2)
	return ret
}

func (l *wordParameterList) getCost(wordId int32) int16 {
	_, ret := bufferToInt16(l.bytebuffer, l.offset+wordParameterListElementSize*int(wordId)+4)
	return ret
}

func (l *wordParameterList) setCost(wordId int32, cost int16) {
	if l.isCopied {
		l.copyBuffer()
	}

	s := l.offset + wordParameterListElementSize*int(wordId) + 4
	binary.LittleEndian.PutUint16(l.bytebuffer[s:], uint16(cost))
}

// syncronized ???
func (l *wordParameterList) copyBuffer() {
	nl := int(wordParameterListElementSize) * int(l.size)
	newBuffer := make([]byte, nl, nl)
	s := l.offset
	copy(newBuffer, l.bytebuffer[s:s+nl])
	l.bytebuffer = newBuffer
	l.offset = 0
	l.isCopied = true
}

type wordInfoList struct {
	bytebuffer      []byte
	offset          int
	wordSize        int32
	bufferToStringF bufferToStringFunc
}

func newWordInfoList(bytebuffer []byte, offset int, wordSize int32, bufferToStringF bufferToStringFunc) *wordInfoList {
	return &wordInfoList{
		bytebuffer:      bytebuffer,
		offset:          offset,
		wordSize:        wordSize,
		bufferToStringF: bufferToStringF,
	}
}

func (l *wordInfoList) getWordInfo(wordId int32) *WordInfo {
	index := l.wordIdToOffset(wordId)

	index, surface := l.bufferToStringF(l.bytebuffer, index)
	headwordLength := int16(l.bytebuffer[index])
	index += 1
	index, posId := bufferToInt16(l.bytebuffer, index)
	index, normalizedForm := l.bufferToStringF(l.bytebuffer, index)
	if normalizedForm == "" {
		normalizedForm = surface
	}
	index, dictionaryFormWordId := bufferToInt32(l.bytebuffer, index)
	index, readingForm := l.bufferToStringF(l.bytebuffer, index)
	if readingForm == "" {
		readingForm = surface
	}
	index, aUnitSplit := bufferToInt32Array(l.bytebuffer, index)
	if !l.isValidSplit(aUnitSplit) {
		aUnitSplit = make([]int32, 0)
	}

	index, bUnitSplit := bufferToInt32Array(l.bytebuffer, index)
	if !l.isValidSplit(aUnitSplit) {
		bUnitSplit = make([]int32, 0)
	}

	index, wordStructure := bufferToInt32Array(l.bytebuffer, index)
	if !l.isValidSplit(aUnitSplit) {
		aUnitSplit = make([]int32, 0)
	}

	dictionaryForm := surface
	if dictionaryFormWordId >= 0 && dictionaryFormWordId != wordId {
		wi := l.getWordInfo(dictionaryFormWordId)
		dictionaryForm = wi.Surface
	}

	return &WordInfo{
		Surface:              surface,
		HeadwordLength:       headwordLength,
		PosId:                posId,
		NormalizedForm:       normalizedForm,
		DictionaryFormWordId: dictionaryFormWordId,
		DictionaryForm:       dictionaryForm,
		ReadingForm:          readingForm,
		AUnitSplit:           aUnitSplit,
		BUnitSplit:           bUnitSplit,
		WordStructure:        wordStructure,
	}
}

func (l *wordInfoList) wordIdToOffset(wordId int32) int {
	s := l.offset + 4*int(wordId)
	_, ret := bufferToInt32(l.bytebuffer, s)
	return int(ret)
}

func (l *wordInfoList) isValidSplit(split []int32) bool {
	for _, wordId := range split {
		if wordId >= l.wordSize {
			return false
		}
	}
	return true
}

type DoubleArrayLexicon struct {
	wordIdT    *wordIdTable
	wordParams *wordParameterList
	wordInfos  *wordInfoList
	trie       *dartsclone.DoubleArray
}

func NewDoubleArrayLexicon(bytebuffer []byte, offset int, utf16string bool) *DoubleArrayLexicon {
	var size uint32
	trie := dartsclone.NewDoubleArray()
	offset, size = bufferToUint32(bytebuffer, offset)
	trie.SetBuffer(bytebuffer[offset : offset+int(size)*4])
	offset += trie.TotalSize()

	wordIdT := newWordIdTable(bytebuffer, offset)
	offset += wordIdT.storageSize()

	wordParams := newWordParameterList(bytebuffer, offset)
	offset += wordParams.storageSize()

	var wordInfos *wordInfoList
	if utf16string {
		wordInfos = newWordInfoList(bytebuffer, offset, wordParams.size, bufferToStringUtf16)
	} else {
		wordInfos = newWordInfoList(bytebuffer, offset, wordParams.size, bufferToString)
	}

	return &DoubleArrayLexicon{
		wordIdT:    wordIdT,
		wordParams: wordParams,
		wordInfos:  wordInfos,
		trie:       trie,
	}
}

func (lexicon *DoubleArrayLexicon) Lookup(text []byte, offset int) *DoubleArrayLexiconIterator {
	it := lexicon.trie.CommonPrefixSearchItr(text, offset)
	return newDoubleArrayLexiconIterator(it, lexicon.wordIdT)
}

func (lexicon *DoubleArrayLexicon) GetLeftId(wordId int32) int16 {
	return lexicon.wordParams.getLeftId(wordId)
}

func (lexicon *DoubleArrayLexicon) GetRightId(wordId int32) int16 {
	return lexicon.wordParams.getRightId(wordId)
}

func (lexicon *DoubleArrayLexicon) GetCost(wordId int32) int16 {
	return lexicon.wordParams.getCost(wordId)
}

func (lexicon *DoubleArrayLexicon) GetWordInfo(wordId int32) *WordInfo {
	return lexicon.wordInfos.getWordInfo(wordId)
}

func (lexicon *DoubleArrayLexicon) GetDictionaryId(wordId int32) int {
	return 0
}

func (lexicon *DoubleArrayLexicon) Size() int32 {
	return lexicon.wordParams.size
}

const maxint16 = int16(^uint16(0) >> 1)
const minint16 = -maxint16 - 1

type CalculateCostFunc func(text string) (int16, error)

func (lexicon *DoubleArrayLexicon) CalculateCost(cf CalculateCostFunc) error {
	var wordId int32
	for ; wordId < lexicon.wordParams.size; wordId++ {
		if lexicon.wordParams.getCost(wordId) != minint16 {
			continue
		}
		wi := lexicon.wordInfos.getWordInfo(wordId)
		cost, err := cf(wi.Surface)
		if err != nil {
			return err
		}
		lexicon.wordParams.setCost(wordId, cost)
	}
	return nil
}

func (lexicon *DoubleArrayLexicon) WriteTrieTo(writer io.Writer) (int, error) {
	err := binary.Write(writer, binary.LittleEndian, uint32(lexicon.trie.Length()))
	if err != nil {
		return 0, err
	}
	n, err := writer.Write(lexicon.trie.ByteArray())
	if err != nil {
		return 4, err
	}
	return n + 4, nil
}

func (lexicon *DoubleArrayLexicon) WriteWordIdTableTo(writer io.Writer) (int, error) {
	err := binary.Write(writer, binary.LittleEndian, uint32(lexicon.wordIdT.size))
	if err != nil {
		return 0, err
	}
	n, err := writer.Write(lexicon.wordIdT.bytebuffer)
	if err != nil {
		return 4, err
	}
	return n + 4, nil
}

func (lexicon *DoubleArrayLexicon) WriteWordParamsTo(writer io.Writer) (int, error) {
	size := lexicon.wordParams.size
	err := binary.Write(writer, binary.LittleEndian, uint32(size))
	if err != nil {
		return 0, err
	}
	n, err := writer.Write(lexicon.wordParams.bytebuffer[lexicon.wordParams.offset : lexicon.wordParams.offset+wordParameterListElementSize*int(size)])
	if err != nil {
		return 4, err
	}
	return n + 4, nil
}

func (lexicon *DoubleArrayLexicon) WriteWordInfos(writer io.Writer, offset int64, offsetlen int64, utf16string bool) (int, *bytes.Buffer, error) {
	var writeStringF writeStringFunc
	if utf16string {
		writeStringF = writeStringUtf16
	} else {
		writeStringF = writeString
	}

	buffer := bytes.NewBuffer([]byte{})

	offsets := bytes.NewBuffer(make([]byte, 0, offsetlen))
	base := offset + offsetlen
	position := base
	for wordId := int32(0); wordId < lexicon.Size(); wordId++ {
		wi := lexicon.GetWordInfo(wordId)
		err := binary.Write(offsets, binary.LittleEndian, uint32(position))
		if err != nil {
			return 0, offsets, err
		}
		err = writeStringF(buffer, wi.Surface)
		if err != nil {
			return 0, offsets, err
		}
		// may overflow
		err = buffer.WriteByte(byte(wi.HeadwordLength))
		if err != nil {
			return 0, offsets, err
		}
		err = binary.Write(buffer, binary.LittleEndian, uint16(wi.PosId))
		if err != nil {
			return 0, offsets, err
		}
		var normalizedForm string
		if wi.NormalizedForm != wi.Surface {
			normalizedForm = wi.NormalizedForm
		}
		err = writeStringF(buffer, normalizedForm)
		if err != nil {
			return 0, offsets, err
		}
		err = binary.Write(buffer, binary.LittleEndian, uint32(wi.DictionaryFormWordId))
		if err != nil {
			return 0, offsets, err
		}
		var readingForm string
		if wi.ReadingForm != wi.Surface {
			readingForm = wi.ReadingForm
		}
		err = writeStringF(buffer, readingForm)
		if err != nil {
			return 0, offsets, err
		}
		err = writeIntArray(buffer, wi.AUnitSplit)
		if err != nil {
			return 0, offsets, err
		}
		err = writeIntArray(buffer, wi.BUnitSplit)
		if err != nil {
			return 0, offsets, err
		}
		err = writeIntArray(buffer, wi.WordStructure)
		if err != nil {
			return 0, offsets, err
		}
		n, err := buffer.WriteTo(writer)
		buffer.Reset()
		position += n
	}
	return int(position - base), offsets, nil
}

type DoubleArrayLexiconIterator struct {
	wordIdT *wordIdTable
	dait    *dartsclone.Iterator
	wordIds []int32
	length  int
	index   int
}

func newDoubleArrayLexiconIterator(dait *dartsclone.Iterator, wordIdT *wordIdTable) *DoubleArrayLexiconIterator {
	return &DoubleArrayLexiconIterator{
		wordIdT: wordIdT,
		dait:    dait,
		index:   -1,
	}
}

func (it *DoubleArrayLexiconIterator) Next() bool {
	if it.dait.Err() != nil {
		return false
	}
	if it.index < 0 {
		return it.dait.Next()
	} else {
		return it.index < len(it.wordIds) || it.dait.Next()
	}
}

func (it *DoubleArrayLexiconIterator) Get() (int32, int) {
	if it.index < 0 || it.index >= len(it.wordIds) {
		tindex, length := it.dait.Get()
		if it.dait.Err() != nil {
			return -1, 0
		}
		it.wordIds = it.wordIdT.get(tindex)
		it.length = length
		it.index = 0
	}
	wordId := it.wordIds[it.index]
	it.index++
	return wordId, it.length
}

func (it *DoubleArrayLexiconIterator) Err() error {
	return it.dait.Err()
}
