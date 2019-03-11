package gosudachi

import (
	"fmt"

	"github.com/msnoigrs/gosudachi/dictionary"
)

type JoinKatakanaOovPluginConfig struct {
	OovPOS    *[]string
	minLength *int
}

type JoinKatakanaOovPlugin struct {
	config    *JoinKatakanaOovPluginConfig
	oovPosId  int16
	minLength int
}

func NewJoinKatakanaOovPlugin(config *JoinKatakanaOovPluginConfig) *JoinKatakanaOovPlugin {
	if config == nil {
		config = &JoinKatakanaOovPluginConfig{}
	}
	return &JoinKatakanaOovPlugin{
		config: config,
	}
}

func (p *JoinKatakanaOovPlugin) GetConfigStruct() interface{} {
	if p.config == nil {
		p.config = &JoinKatakanaOovPluginConfig{}
	}
	return p.config
}

func (p *JoinKatakanaOovPlugin) SetUp(grammar *dictionary.Grammar) error {
	if p.config.OovPOS == nil || len(*p.config.OovPOS) == 0 {
		return fmt.Errorf("JoinKatakanaOovPlugin: oovPOS is not specified")
	}
	p.oovPosId = grammar.GetPartOfSpeechId(*p.config.OovPOS)
	if p.oovPosId < 0 {
		return fmt.Errorf("JoinKatakanaOovPlugin: oovPOS is invalid")
	}
	minLength := 1
	if p.config.minLength != nil {
		minLength = *p.config.minLength
		if minLength < 0 {
			return fmt.Errorf("JoinKatakanaOovPlugin: minLength is negative")
		}
	}
	p.minLength = minLength
	p.config = nil
	return nil
}

func isShorter(length int, text *InputText, node *LatticeNode) bool {
	return text.CodePointCount(node.Begin, node.End) < length
}

func isKatakanaNode(text *InputText, node *LatticeNode) bool {
	types := GetCharCategoryTypes(text, node)
	return (types & dictionary.KATAKANA) == dictionary.KATAKANA
}

func canOovBowNode(text *InputText, node *LatticeNode) bool {
	types := GetCharCategoryTypes(text, node)
	return types&dictionary.NOOOVBOW != dictionary.NOOOVBOW
}

func (p *JoinKatakanaOovPlugin) Rewrite(text *InputText, path *[]*LatticeNode, lattice *Lattice) error {
	for i := 0; i < len(*path); i++ {
		node := (*path)[i]
		if (node.IsOov || isShorter(p.minLength, text, node)) &&
			isKatakanaNode(text, node) {
			begin := i - 1
			for ; begin >= 0; begin-- {
				if !isKatakanaNode(text, (*path)[begin]) {
					begin++
					break
				}
			}
			if begin < 0 {
				begin = 0
			}
			end := i + 1
			for ; end < len(*path); end++ {
				if !isKatakanaNode(text, (*path)[end]) {
					break
				}
			}
			for begin != end && !canOovBowNode(text, (*path)[begin]) {
				begin++
			}
			if end-begin > 1 {
				_, err := ConcatenateOov(path, begin, end, p.oovPosId, lattice)
				if err != nil {
					return fmt.Errorf("JoinKatakanaOovPlugin: %s", err)
				}
				i = begin + 1
			}
		}
	}
	return nil
}
