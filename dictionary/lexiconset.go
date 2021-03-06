package dictionary

const (
	LexiconSetMaxDictionaries = 16
)

type LexiconSet struct {
	lexicons   []*DoubleArrayLexicon
	posOffsets []int32
}

func NewLexiconSet(systemLexicon *DoubleArrayLexicon) *LexiconSet {
	return &LexiconSet{
		lexicons:   []*DoubleArrayLexicon{systemLexicon},
		posOffsets: []int32{0},
	}
}

func (s *LexiconSet) Add(lexicon *DoubleArrayLexicon, posOffset int32) {
	s.lexicons = append(s.lexicons, lexicon)
	s.posOffsets = append(s.posOffsets, posOffset)
}

func (s *LexiconSet) IsFull() bool {
	return len(s.lexicons) >= LexiconSetMaxDictionaries
}

func (s *LexiconSet) Lookup(text []byte, offset int) *LexiconSetIterator {
	return newLexiconSetIterator(text, offset, s.lexicons)
}

func (s *LexiconSet) GetWordId(headword string, posId int16, readingForm string) int32 {
	for dictId := 1; dictId < len(s.lexicons); dictId++ {
		wordId := s.lexicons[dictId].GetWordId(headword, posId, readingForm)
		if wordId >= 0 {
			// buildWordId
			return int32(uint32(dictId)<<28) | wordId
		}
	}
	return s.lexicons[0].GetWordId(headword, posId, readingForm)
}

func (s *LexiconSet) GetLeftId(wordId int32) int16 {
	dictId := int(uint32(wordId) >> 28)
	wordId = int32(uint32(wordId) & 0xfffffff)
	return s.lexicons[dictId].GetLeftId(wordId)
}

func (s *LexiconSet) GetRightId(wordId int32) int16 {
	dictId := int(uint32(wordId) >> 28)
	wordId = int32(uint32(wordId) & 0xfffffff)
	return s.lexicons[dictId].GetRightId(wordId)
}

func (s *LexiconSet) GetCost(wordId int32) int16 {
	dictId := int(uint32(wordId) >> 28)
	wordId = int32(uint32(wordId) & 0xfffffff)
	return s.lexicons[dictId].GetCost(wordId)
}

func (s *LexiconSet) GetWordInfo(wordId int32) *WordInfo {
	dictId := int(uint32(wordId) >> 28)
	wordId = int32(uint32(wordId) & 0xfffffff)
	wi := s.lexicons[dictId].GetWordInfo(wordId)
	if dictId > 0 && int32(wi.PosId) >= s.posOffsets[1] {
		// user defined part-of-speech
		wi.PosId = int16(int32(wi.PosId) - s.posOffsets[1] + s.posOffsets[dictId])
	}
	s.convertSplit(wi.AUnitSplit, dictId)
	s.convertSplit(wi.BUnitSplit, dictId)
	s.convertSplit(wi.WordStructure, dictId)
	return wi
}

func (s *LexiconSet) GetDictionaryId(wordId int32) int {
	return int(uint32(wordId) >> 28)
}

func (s *LexiconSet) Size() int32 {
	var n int32
	for _, l := range s.lexicons {
		n += l.Size()
	}
	return n
}

func (s *LexiconSet) convertSplit(split []int32, dictId int) {
	for i, id := range split {
		if s.GetDictionaryId(id) > 0 {
			wordId := uint32(id) & 0xfffffff
			// buildWordId
			split[i] = int32(uint32(dictId<<28) | wordId)
		}
	}
}

type LexiconSetIterator struct {
	text     []byte
	offset   int
	dictId   int
	lexicons []*DoubleArrayLexicon
	dalit    *DoubleArrayLexiconIterator
}

func newLexiconSetIterator(text []byte, offset int, lexicons []*DoubleArrayLexicon) *LexiconSetIterator {
	var (
		dalit  *DoubleArrayLexiconIterator
		dictId int
	)
	if len(lexicons) == 1 {
		dictId = 0
	} else {
		dictId = 1
	}
	dalit = lexicons[dictId].Lookup(text, offset)

	return &LexiconSetIterator{
		text:     text,
		offset:   offset,
		dictId:   dictId,
		lexicons: lexicons,
		dalit:    dalit,
	}
}

func (it *LexiconSetIterator) Next() bool {
	if it.dalit.Err() != nil {
		return false
	}
	for !it.dalit.Next() {
		if it.dictId == 0 {
			return false
		}
		it.dictId++
		if it.dictId >= len(it.lexicons) {
			it.dictId = 0
		}
		it.dalit = it.lexicons[it.dictId].Lookup(it.text, it.offset)
	}
	return true
}

func (it *LexiconSetIterator) Get() (int32, int) {
	rvalue, roffset := it.dalit.Get()
	if it.dalit.Err() != nil {
		return -1, 0
	}
	if it.dictId > 0 {
		// buildWordId
		rvalue = int32(uint32(it.dictId<<28) | uint32(rvalue))
	}
	return rvalue, roffset
}

func (it *LexiconSetIterator) Err() error {
	return it.dalit.Err()
}
