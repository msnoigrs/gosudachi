package gosudachi

import (
	"github.com/msnoigrs/gosudachi/dictionary"
)

type Morpheme struct {
	list     *MorphemeList
	index    int
	wordInfo *dictionary.WordInfo
}

func newMorpheme(list *MorphemeList, index int) *Morpheme {
	return &Morpheme{
		list:  list,
		index: index,
	}
}

func (m *Morpheme) Begin() int {
	return m.list.GetBegin(m.index)
}

func (m *Morpheme) End() int {
	return m.list.GetEnd(m.index)
}

func (m *Morpheme) Surface() string {
	return m.list.GetSurface(m.index)
}

func (m *Morpheme) PartOfSpeech() ([]string, error) {
	wi, err := m.GetWordInfo()
	if err != nil {
		return []string{}, err
	}
	return m.list.grammar.GetPartOfSpeechString(wi.PosId), nil
}

func (m *Morpheme) DictionaryForm() (string, error) {
	wi, err := m.GetWordInfo()
	if err != nil {
		return "", err
	}
	return wi.DictionaryForm, nil
}

func (m *Morpheme) NormalizedForm() (string, error) {
	wi, err := m.GetWordInfo()
	if err != nil {
		return "", err
	}
	return wi.NormalizedForm, nil
}

func (m *Morpheme) ReadingForm() (string, error) {
	wi, err := m.GetWordInfo()
	if err != nil {
		return "", err
	}
	return wi.ReadingForm, nil
}

func (m *Morpheme) Split(mode string) (*MorphemeList, error) {
	wi, err := m.GetWordInfo()
	if err != nil {
		return nil, nil
	}
	return m.list.Split(mode, m.index, wi)
}

func (m *Morpheme) IsOOV() bool {
	return m.list.IsOOV(m.index)
}

func (m *Morpheme) GetWordId() int {
	return m.list.GetWordId(m.index)
}

func (m *Morpheme) GetDictionaryId() int {
	return m.list.GetDictionaryId(m.index)
}

func (m *Morpheme) GetWordInfo() (*dictionary.WordInfo, error) {
	if m.wordInfo == nil {
		wordInfo, err := m.list.GetWordInfo(m.index)
		if err != nil {
			return nil, err
		}
		m.wordInfo = wordInfo
	}
	return m.wordInfo, nil
}

type MorphemeList struct {
	inputText *InputText
	grammar   *dictionary.Grammar
	lexicon   *dictionary.LexiconSet
	path      []*LatticeNode
}

func NewMorphemeList(inputText *InputText, grammar *dictionary.Grammar, lexicon *dictionary.LexiconSet, path []*LatticeNode) *MorphemeList {
	return &MorphemeList{
		inputText: inputText,
		grammar:   grammar,
		lexicon:   lexicon,
		path:      path,
	}
}

func (l *MorphemeList) Length() int {
	return len(l.path)
}

func (l *MorphemeList) Get(index int) *Morpheme {
	return newMorpheme(l, index)
}

func (l *MorphemeList) GetBegin(index int) int {
	return l.path[index].Begin
}

func (l *MorphemeList) GetEnd(index int) int {
	return l.path[index].End
}

func (l *MorphemeList) GetSurface(index int) string {
	node := l.path[index]
	return string([]byte(l.inputText.OriginalText)[node.Begin:node.End])
}

func (l *MorphemeList) GetWordInfo(index int) (*dictionary.WordInfo, error) {
	return l.path[index].GetWordInfo()
}

func (l *MorphemeList) Split(mode string, index int, wi *dictionary.WordInfo) (*MorphemeList, error) {
	var wordIds []int32
	switch mode {
	case "A":
		wordIds = wi.AUnitSplit
	case "B":
		wordIds = wi.BUnitSplit
	default:
		return NewMorphemeList(l.inputText, l.grammar, l.lexicon, []*LatticeNode{l.path[index]}), nil
	}
	if len(wordIds) == 0 || len(wordIds) == 1 {
		return NewMorphemeList(l.inputText, l.grammar, l.lexicon, []*LatticeNode{l.path[index]}), nil
	}

	offset := l.path[index].Begin
	nodes := make([]*LatticeNode, len(wordIds), len(wordIds))
	for i, wid := range wordIds {
		n := NewLatticeNode(l.lexicon, 0, 0, 0, wid)
		n.Begin = offset
		wi, err := n.GetWordInfo()
		if err != nil {
			return nil, err
		}
		offset += int(wi.HeadwordLength)
		n.End = offset
		nodes[i] = n
	}

	return NewMorphemeList(l.inputText, l.grammar, l.lexicon, nodes), nil
}

func (l *MorphemeList) IsOOV(index int) bool {
	return l.path[index].IsOOV()
}

func (l *MorphemeList) GetWordId(index int) int {
	return l.path[index].GetWordId()
}

func (l *MorphemeList) GetDictionaryId(index int) int {
	return l.path[index].GetDictionaryId()
}

func (l *MorphemeList) GetInternalCost() int {
	return l.path[len(l.path)-1].GetPathCost() - l.path[0].GetPathCost()
}
