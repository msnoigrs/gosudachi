package dictionary

const (
	LexiconSetMaxDictionaries = 16
)

type LexiconSet struct {
	lexicons []*DoubleArrayLexicon
}

func NewLexiconSet(systemLexicon *DoubleArrayLexicon) *LexiconSet {
	return &LexiconSet{
		lexicons: []*DoubleArrayLexicon{systemLexicon},
	}
}

func (s *LexiconSet) Add(lexicon *DoubleArrayLexicon) {
	s.lexicons = append(s.lexicons, lexicon)
}

func (s *LexiconSet) IsFull() bool {
	return len(s.lexicons) >= LexiconSetMaxDictionaries
}

func (s *LexiconSet) Lookup(text []byte, offset int) *LexiconSetIterator {
	return newLexiconSetIterator(text, offset, s.lexicons)
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
	return s.lexicons[dictId].GetWordInfo(wordId)
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
