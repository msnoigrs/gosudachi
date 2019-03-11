package gosudachi

import (
	"github.com/msnoigrs/gosudachi/dictionary"
)

type InhibitConnectionPlugin struct {
	inhibitedPair []*[]int
}

func NewInhibitConnectionPlugin(inhibitedPair []*[]int) *InhibitConnectionPlugin {
	return &InhibitConnectionPlugin{
		inhibitedPair: inhibitedPair,
	}
}

func (p *InhibitConnectionPlugin) GetConfigStruct() interface{} {
	return p
}

func (p *InhibitConnectionPlugin) SetUp(grammar *dictionary.Grammar) error {
	return nil
}

func (p *InhibitConnectionPlugin) Edit(grammar *dictionary.Grammar) error {
	for _, pair := range p.inhibitedPair {
		if len(*pair) < 2 {
			continue
		}
		InhibitConnection(grammar, int16((*pair)[0]), int16((*pair)[1]))
	}
	return nil
}
