package gosudachi

import (
	"fmt"

	"github.com/msnoigrs/gosudachi/dictionary"
)

type Settings interface {
	GetBaseConfig() *BaseConfig
}

type BaseConfig struct {
	SystemDict              string
	CharacterDefinitionFile string
	UserDict                []string
	Utf16String             bool
}

type PluginMaker interface {
	GetInputTextPluginArray(f MakeInputTextPluginFunc) ([]InputTextPlugin, error)
	GetOovProviderPluginArray(f MakeOovProviderPluginFunc) ([]OovProviderPlugin, error)
	GetPathRewritePluginArray(f MakePathRewritePluginFunc) ([]PathRewritePlugin, error)
	GetEditConnectionCostPluginArray(f MakeEditConnectionCostPluginFunc) ([]EditConnectionCostPlugin, error)
}

type Plugin interface {
	GetConfigStruct() interface{}
}

type MakeInputTextPluginFunc func(n string) InputTextPlugin
type MakeEditConnectionCostPluginFunc func(n string) EditConnectionCostPlugin
type MakeOovProviderPluginFunc func(n string) OovProviderPlugin
type MakePathRewritePluginFunc func(n string) PathRewritePlugin

func DefMakeInputTextPlugin(k string) InputTextPlugin {
	switch k {
	case "DefaultInputTextPlugin", "com.worksap.nlp.sudachi.DefaultInputTextPlugin":
		return NewDefaultInputTextPlugin(nil)
	case "ProlongedSoundMarkInputTextPlugin", "com.worksap.nlp.sudachi.ProlongedSoundMarkInputTextPlugin":
		return NewProlongedSoundMarkInputTextPlugin(nil)
	}
	return nil
}

func DefMakeEditConnectionCostPlugin(k string) EditConnectionCostPlugin {
	switch k {
	case "InhibitConnectionPlugin", "com.worksap.nlp.sudachi.InhibitConnectionPlugin":
		return NewInhibitConnectionPlugin([]*[]int{})
	}
	return nil
}

func DefMakeOovProviderPlugin(k string) OovProviderPlugin {
	switch k {
	case "MeCabOovProviderPlugin", "com.worksap.nlp.sudachi.MeCabOovProviderPlugin":
		return NewMeCabOovProviderPlugin(nil)
	case "SimpleOovProviderPlugin", "com.worksap.nlp.sudachi.SimpleOovProviderPlugin":
		return NewSimpleOovProviderPlugin(nil)
	}
	return nil
}

func DefMakePathRewritePlugin(k string) PathRewritePlugin {
	switch k {
	case "JoinNumericPlugin", "com.worksap.nlp.sudachi.JoinNumericPlugin":
		return NewJoinNumericPlugin(nil)
	case "JoinKatakanaOovPlugin", "com.worksap.nlp.sudachi.JoinKatakanaOovPlugin":
		return NewJoinKatakanaOovPlugin(nil)
	}
	return nil
}

type EditConnectionCostPlugin interface {
	Plugin
	SetUp(grammar *dictionary.Grammar) error
	Edit(grammar *dictionary.Grammar) error
}

func InhibitConnection(grammar *dictionary.Grammar, leftId int16, rightId int16) {
	grammar.SetConnectCost(leftId, rightId, dictionary.InhibitedConnection)
}

type PathRewritePlugin interface {
	Plugin
	SetUp(grammar *dictionary.Grammar) error
	Rewrite(text *InputText, path *[]*LatticeNode, lattice *Lattice) error
}

func ConcatenateNodes(path *[]*LatticeNode, begin int, end int, lattice *Lattice, normalizedForm string) (*LatticeNode, error) {
	if begin >= end {
		return nil, fmt.Errorf("begin >= end")
	}
	tpath := *path
	b := tpath[begin].Begin
	e := tpath[end-1].End
	bwi := tpath[begin].GetWordInfo()
	posId := bwi.PosId
	var (
		surfaceLen        int
		normalizedFormLen int
		dictionaryFormLen int
		readingFormLen    int
		length            int16
	)
	wilist := make([]*dictionary.WordInfo, len(tpath), len(tpath))
	for i, n := range tpath {
		info := n.GetWordInfo()
		wilist[i] = info
		surfaceLen += len(info.Surface)
		length += info.HeadwordLength
		if normalizedForm == "" {
			normalizedFormLen += len(info.NormalizedForm)
		}
		dictionaryFormLen += len(info.DictionaryForm)
		readingFormLen += len(info.ReadingForm)
	}
	csurface := make([]byte, 0, surfaceLen)
	var cnormalizedForm []byte
	if normalizedForm == "" {
		cnormalizedForm = make([]byte, 0, normalizedFormLen)
	}
	cdictionaryForm := make([]byte, 0, dictionaryFormLen)
	creadingForm := make([]byte, 0, readingFormLen)
	for _, wi := range wilist {
		csurface = append(csurface, []byte(wi.Surface)...)
		if normalizedForm == "" {
			cnormalizedForm = append(cnormalizedForm, []byte(wi.NormalizedForm)...)
		}
		cdictionaryForm = append(cdictionaryForm, []byte(wi.DictionaryForm)...)
		creadingForm = append(creadingForm, []byte(wi.ReadingForm)...)
	}
	if normalizedForm == "" {
		normalizedForm = string(cnormalizedForm)
	}
	wi := &dictionary.WordInfo{
		Surface:        string(csurface),
		HeadwordLength: length,
		PosId:          posId,
		NormalizedForm: normalizedForm,
		DictionaryForm: string(cdictionaryForm),
		ReadingForm:    string(creadingForm),
	}

	node := &LatticeNode{}
	node.SetRange(b, e)
	node.SetWordInfo(wi)
	*path = replaceNode(tpath, begin, end, node)
	return node, nil
}

func ConcatenateOov(path *[]*LatticeNode, begin int, end int, posId int16, lattice *Lattice) (*LatticeNode, error) {
	if begin >= end {
		return nil, fmt.Errorf("begin >= end")
	}
	tpath := *path
	b := tpath[begin].Begin
	e := tpath[end-1].End

	n := lattice.GetMinimumNode(b, e)
	if n != nil {
		*path = replaceNode(tpath, begin, end, n)
		return n, nil
	}

	var (
		surfaceLen int
		length     int16
	)
	wilist := make([]*dictionary.WordInfo, len(tpath), len(tpath))
	for i, n := range tpath {
		info := n.GetWordInfo()
		wilist[i] = info
		surfaceLen += len(info.Surface)
		length += info.HeadwordLength
	}
	csurface := make([]byte, 0, surfaceLen)
	for _, wi := range wilist {
		csurface = append(csurface, []byte(wi.Surface)...)
	}
	s := string(csurface)
	wi := &dictionary.WordInfo{
		Surface:        s,
		HeadwordLength: length,
		PosId:          posId,
		NormalizedForm: s,
		DictionaryForm: s,
		ReadingForm:    "",
	}

	node := &LatticeNode{}
	node.SetRange(b, e)
	node.SetWordInfo(wi)
	node.IsOov = true
	*path = replaceNode(tpath, begin, end, node)
	return node, nil
}

func GetCharCategoryTypes(text *InputText, node *LatticeNode) uint32 {
	return text.GetCharCategoryTypesRange(node.Begin, node.End)
}

func replaceNode(path []*LatticeNode, begin int, end int, node *LatticeNode) []*LatticeNode {
	d := end - begin
	if d > 1 && end < len(path) {
		copy(path[begin+1:], path[end:])
		path = path[:len(path)-d+1]
	}
	path[begin] = node
	return path
}

type InputTextPlugin interface {
	Plugin
	SetUp() error
	Rewrite(builder *InputTextBuilder) error
}

type OovProviderPlugin interface {
	Plugin
	SetUp(grammar *dictionary.Grammar) error
	ProvideOOV(inputText *InputText, offset int, hasOtherWords bool) ([]*LatticeNode, error)
}

func GetOOV(p OovProviderPlugin, inputText *InputText, offset int, hasOtherWords bool) ([]*LatticeNode, error) {
	nodes, err := p.ProvideOOV(inputText, offset, hasOtherWords)
	if err != nil {
		return []*LatticeNode{}, err
	}
	for _, node := range nodes {
		wi := node.GetWordInfo()
		node.Begin = offset
		node.End = offset + int(wi.HeadwordLength)
	}
	return nodes, nil
}

func CreateNodeOfOOV() *LatticeNode {
	return &LatticeNode{
		IsOov: true,
	}
}
