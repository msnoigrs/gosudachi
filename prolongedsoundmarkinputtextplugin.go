package gosudachi

import (
	"fmt"
)

type ProlongedSoundMarkInputTextPluginConfig struct {
	ProlongedSoundMarks *[]string
	ReplacementSymbol   *string
}

type ProlongedSoundMarkInputTextPlugin struct {
	config                *ProlongedSoundMarkInputTextPluginConfig
	prolongedSoundMarkMap map[rune]bool
	replacementSymbol     []rune
}

func NewProlongedSoundMarkInputTextPlugin(config *ProlongedSoundMarkInputTextPluginConfig) *ProlongedSoundMarkInputTextPlugin {
	if config == nil {
		config = &ProlongedSoundMarkInputTextPluginConfig{}
	}
	return &ProlongedSoundMarkInputTextPlugin{
		config:                config,
		prolongedSoundMarkMap: map[rune]bool{},
	}
}

func (p *ProlongedSoundMarkInputTextPlugin) GetConfigStruct() interface{} {
	if p.config == nil {
		p.config = &ProlongedSoundMarkInputTextPluginConfig{}
	}
	return p.config
}

func (p *ProlongedSoundMarkInputTextPlugin) SetUp() error {
	if p.config.ProlongedSoundMarks == nil || len(*p.config.ProlongedSoundMarks) == 0 {
		return fmt.Errorf("ProlongedSoundMarkInputTextPlugin: prolongedSoundMarkStrings is not specified")
	}
	if p.config.ReplacementSymbol == nil {
		return fmt.Errorf("ProlongedSoundMarkInputTextPlugin: replacementSymbol is not specified")
	}
	if p.prolongedSoundMarkMap == nil {
		p.prolongedSoundMarkMap = map[rune]bool{}
	}
	for _, s := range *p.config.ProlongedSoundMarks {
		runes := []rune(s)
		if len(runes) > 0 {
			p.prolongedSoundMarkMap[runes[0]] = true
		}
	}
	p.replacementSymbol = []rune(*p.config.ReplacementSymbol)
	p.config = nil
	return nil
}

func (p *ProlongedSoundMarkInputTextPlugin) Rewrite(builder *InputTextBuilder) error {
	runes := builder.GetText()

	runelen := len(runes)
	offset := 0
	markStartIndex := runelen
	isProlongedSoundMark := false
	for i := 0; i < runelen; i++ {
		_, ok := p.prolongedSoundMarkMap[runes[i]]
		if !isProlongedSoundMark && ok {
			isProlongedSoundMark = true
			markStartIndex = i
		} else if isProlongedSoundMark && !ok {
			if (i - markStartIndex) > 1 {
				builder.Replace(markStartIndex-offset, i-offset, p.replacementSymbol)
				offset += i - markStartIndex - 1
			}
			isProlongedSoundMark = false
		}
	}
	if isProlongedSoundMark && (runelen-markStartIndex) > 1 {
		builder.Replace(markStartIndex-offset, runelen-offset, p.replacementSymbol)
	}
	return nil
}
