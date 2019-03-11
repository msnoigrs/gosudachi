package gosudachi

import (
	"fmt"

	"github.com/msnoigrs/gosudachi/dictionary"
)

type SimpleOovProviderPluginConfig struct {
	OovPos  *[]string
	LeftId  *int16
	RightId *int16
	Cost    *int16
}

type SimpleOovProviderPlugin struct {
	config   *SimpleOovProviderPluginConfig
	oovPosId int16
	leftId   int16
	rightId  int16
	cost     int16
}

func NewSimpleOovProviderPlugin(config *SimpleOovProviderPluginConfig) *SimpleOovProviderPlugin {
	if config == nil {
		config = &SimpleOovProviderPluginConfig{}
	}
	return &SimpleOovProviderPlugin{
		config: config,
	}
}

func (p *SimpleOovProviderPlugin) GetConfigStruct() interface{} {
	if p.config == nil {
		p.config = &SimpleOovProviderPluginConfig{}
	}
	return p.config
}

func (p *SimpleOovProviderPlugin) SetUp(grammar *dictionary.Grammar) error {
	if p.config.OovPos == nil {
		return fmt.Errorf("SimpleOovProviderPlugin: oovPOS is not specified")
	}
	if p.config.LeftId == nil {
		return fmt.Errorf("SimpleOovProviderPlugin: leftId is not specified")
	}
	if p.config.RightId == nil {
		return fmt.Errorf("SimpleOovProviderPlugin: rightId is not specified")
	}
	if p.config.Cost == nil {
		return fmt.Errorf("SimpleOovProviderPlugin: cost is not specified")
	}
	if len(*(p.config.OovPos)) == 0 {
		return fmt.Errorf("SimpleOovProviderPlugin: oovPOS is zero length")
	}
	oovPosId := grammar.GetPartOfSpeechId(*p.config.OovPos)
	if oovPosId < 0 {
		return fmt.Errorf("SimpleOovProviderPlugin: oovPOS is invalid")
	}
	p.oovPosId = oovPosId
	p.leftId = *p.config.LeftId
	p.rightId = *p.config.RightId
	p.cost = *p.config.Cost
	p.config = nil
	return nil
}

func (p *SimpleOovProviderPlugin) ProvideOOV(inputText *InputText, offset int, hasOtherWords bool) ([]*LatticeNode, error) {
	if !hasOtherWords {
		node := CreateNodeOfOOV()
		node.SetParameter(p.leftId, p.rightId, p.cost)
		length := inputText.GetCodePointsOffsetLength(offset, 1)
		s := inputText.GetSubstring(offset, offset+length)
		wi := &dictionary.WordInfo{
			Surface:        s,
			HeadwordLength: int16(length),
			PosId:          p.oovPosId,
			NormalizedForm: s,
			DictionaryForm: s,
			ReadingForm:    "",
		}
		node.SetWordInfo(wi)
		return []*LatticeNode{node}, nil
	} else {
		return []*LatticeNode{}, nil
	}
}
