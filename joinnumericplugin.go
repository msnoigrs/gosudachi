package gosudachi

import (
	"fmt"

	"github.com/msnoigrs/gosudachi/dictionary"
)

type JoinNumericPluginConfig struct {
	EnableNormalize *bool
}

type JoinNumericPlugin struct {
	config          *JoinNumericPluginConfig
	enableNormalize bool
	numericPosId    int16
}

func NewJoinNumericPlugin(config *JoinNumericPluginConfig) *JoinNumericPlugin {
	if config == nil {
		config = &JoinNumericPluginConfig{}
	}
	return &JoinNumericPlugin{
		config: config,
	}
}

func (p *JoinNumericPlugin) GetConfigStruct() interface{} {
	if p.config == nil {
		p.config = &JoinNumericPluginConfig{}
	}
	return p.config
}

func (p *JoinNumericPlugin) SetUp(grammar *dictionary.Grammar) error {
	p.numericPosId = grammar.GetPartOfSpeechId(NumericPos)
	if p.config.EnableNormalize == nil {
		p.enableNormalize = true
	} else {
		p.enableNormalize = *p.config.EnableNormalize
	}
	p.config = nil
	return nil
}

func (p *JoinNumericPlugin) concatNodes(path *[]*LatticeNode, begin int, end int, lattice *Lattice, parser *numericParser) error {
	tpath := *path
	wi := tpath[begin].GetWordInfo()
	if wi.PosId != p.numericPosId {
		return nil
	}
	if p.enableNormalize {
		normalizedForm := parser.getNormalized()
		if end-begin > 1 ||
			normalizedForm != wi.NormalizedForm {
			_, err := ConcatenateNodes(path, begin, end, lattice, normalizedForm)
			if err != nil {
				return err
			}
		}
	} else {
		if end-begin > 1 {
			_, err := ConcatenateNodes(path, begin, end, lattice, "")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *JoinNumericPlugin) Rewrite(text *InputText, path *[]*LatticeNode, lattice *Lattice) error {
	beginIndex := -1
	commaAsDigit := true
	periodAsDigit := true
	parser := newNumericParser()

	for i := 0; i < len(*path); i++ {
		node := (*path)[i]
		types := GetCharCategoryTypes(text, node)
		wi := node.GetWordInfo()
		s := wi.NormalizedForm
		if (types&dictionary.NUMERIC) == dictionary.NUMERIC ||
			(types&dictionary.KANJINUMERIC) == dictionary.KANJINUMERIC ||
			(periodAsDigit && s == ".") ||
			(commaAsDigit && s == ",") {

			if beginIndex < 0 {
				parser.clear()
				beginIndex = i
			}

			for _, c := range s {
				if !parser.append(c) {
					if beginIndex >= 0 {
						if parser.errorState == errComma {
							commaAsDigit = false
							i = beginIndex - 1
						} else if parser.errorState == errPoint {
							periodAsDigit = false
							i = beginIndex - 1
						}
						beginIndex = -1
					}
					break
				}
			}
		} else {
			if beginIndex >= 0 {
				if parser.done() {
					err := p.concatNodes(path, beginIndex, i, lattice, parser)
					if err != nil {
						return fmt.Errorf("JoinNumericPlugin: %s", err)
					}
					i = beginIndex + 1
				} else {
					wi := (*path)[i-1].GetWordInfo()
					ss := wi.NormalizedForm
					if (parser.errorState == errComma && ss == ",") ||
						(parser.errorState == errPoint && ss == ".") {
						err := p.concatNodes(path, beginIndex, i-1, lattice, parser)
						if err != nil {
							return fmt.Errorf("JoinNumericPlugin: %s", err)
						}
						i = beginIndex + 2
					}
				}
			}
			beginIndex = -1
			if !commaAsDigit && s != "," {
				commaAsDigit = true
			}
			if !periodAsDigit && s != "." {
				periodAsDigit = true
			}
		}
	}

	if beginIndex >= 0 {
		if parser.done() {
			p.concatNodes(path, beginIndex, len(*path), lattice, parser)
		} else {
			wi := (*path)[len(*path)-1].GetWordInfo()
			ss := wi.NormalizedForm
			if (parser.errorState == errComma && ss == ",") ||
				(parser.errorState == errPoint && ss == ".") {
				p.concatNodes(path, beginIndex, len(*path)-1, lattice, parser)
			}
		}
	}
	return nil
}

var NumericPos []string = []string{"名詞", "数詞", "*", "*", "*", "*"}
