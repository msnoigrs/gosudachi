package gosudachi

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/msnoigrs/gosudachi/data"
	"github.com/msnoigrs/gosudachi/dictionary"
	"github.com/msnoigrs/gosudachi/internal/lnreader"
)

type categoryInfo struct {
	catType  uint32
	isInvoke bool
	isGroup  bool
	length   int
}

type oov struct {
	leftId  int16
	rightId int16
	cost    int16
	posId   int16
}

type MeCabOovProviderPluginConfig struct {
	CharDef *string
	UnkDef  *string
}

type MeCabOovProviderPlugin struct {
	config     *MeCabOovProviderPluginConfig
	categories map[uint32]*categoryInfo
	oovList    map[uint32]*[]*oov
}

func NewMeCabOovProviderPlugin(config *MeCabOovProviderPluginConfig) *MeCabOovProviderPlugin {
	if config == nil {
		config = &MeCabOovProviderPluginConfig{}
	}
	return &MeCabOovProviderPlugin{
		config:     config,
		categories: map[uint32]*categoryInfo{},
		oovList:    map[uint32]*[]*oov{},
	}
}

func (p *MeCabOovProviderPlugin) GetConfigStruct() interface{} {
	if p.config == nil {
		p.config = &MeCabOovProviderPluginConfig{}
	}
	return p.config
}

func (p *MeCabOovProviderPlugin) SetUp(grammar *dictionary.Grammar) error {
	if p.config.CharDef == nil {
		zstr := ""
		p.config.CharDef = &zstr
	}
	if p.config.UnkDef == nil {
		zstr := ""
		p.config.UnkDef = &zstr
	}
	if p.categories == nil {
		p.categories = map[uint32]*categoryInfo{}
	}
	if p.oovList == nil {
		p.oovList = map[uint32]*[]*oov{}
	}
	err := p.readCharacterProperty(*p.config.CharDef)
	if err != nil {
		return fmt.Errorf("MeCabOovProviderPlugin: %s", err)
	}
	err = p.readOov(*p.config.UnkDef, grammar)
	if err != nil {
		return fmt.Errorf("MeCabOovProviderPlugin: %s", err)
	}
	p.config = nil
	return nil
}

func (p *MeCabOovProviderPlugin) ProvideOOV(inputText *InputText, offset int, hasOtherWords bool) ([]*LatticeNode, error) {
	nodes := []*LatticeNode{}
	length := inputText.GetCharCategoryContinuousLength(offset)
	if length > 0 {
		catTypes := inputText.GetCharCategoryTypes(offset)
		for t := dictionary.DEFAULT; t <= dictionary.NOOOVBOW; t *= 2 {
			if (catTypes & t) != t {
				continue
			}
			cinfo, ok := p.categories[t]
			if !ok {
				continue
			}
			llength := length
			oovs, ok := p.oovList[t]
			if !ok {
				continue
			}
			if cinfo.isGroup && (cinfo.isInvoke || !hasOtherWords) {
				s := inputText.GetSubstring(offset, offset+length)
				for _, oov := range *oovs {
					nodes = append(nodes, p.getOovNode(s, oov, length))
				}
				llength -= 1
			}
			if cinfo.isInvoke || !hasOtherWords {
				for i := 1; i <= cinfo.length; i++ {
					sublength := inputText.GetCodePointsOffsetLength(offset, i)
					if sublength > llength {
						break
					}
					s := inputText.GetSubstring(offset, offset+sublength)
					for _, oov := range *oovs {
						nodes = append(nodes, p.getOovNode(s, oov, sublength))
					}
				}
			}
		}
	}
	return nodes, nil
}

func (p *MeCabOovProviderPlugin) getOovNode(text string, oov *oov, length int) *LatticeNode {
	node := CreateNodeOfOOV()
	node.SetParameter(oov.leftId, oov.rightId, oov.cost)
	wi := &dictionary.WordInfo{
		Surface:        text,
		HeadwordLength: int16(length),
		PosId:          oov.posId,
		NormalizedForm: text,
		DictionaryForm: text,
		ReadingForm:    "",
	}
	node.SetWordInfo(wi)
	return node
}

func (p *MeCabOovProviderPlugin) readCharacterProperty(charDef string) error {
	var charDefReader io.Reader
	if charDef != "" {
		charDefFd, err := os.OpenFile(charDef, os.O_RDONLY, 0644)
		if err != nil {
			return fmt.Errorf("%s: %s", err, charDef)
		}
		defer charDefFd.Close()
		charDefReader = charDefFd
	} else {
		charDefF, err := data.Assets.Open("char.def")
		if err != nil {
			return fmt.Errorf("%s: (data.Assets)char.def", err)
		}
		defer charDefF.Close()
		charDefReader = charDefF
	}

	r := lnreader.NewLineNumberReader(charDefReader)
	for {
		line, err := r.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if lnreader.IsSkipLine(line) {
			continue
		}
		cols := strings.Fields(string(line))
		if len(cols) < 2 {
			return fmt.Errorf("char.def: invalid format at line %d", r.NumLine)
		}
		if strings.HasPrefix(cols[0], "0x") {
			continue
		}
		catType, err := dictionary.GetCategoryType(cols[0])
		if err != nil {
			return fmt.Errorf("char.def: %s is invalid type at line %d", cols[0], r.NumLine)
		}
		_, ok := p.categories[catType]
		if ok {
			return fmt.Errorf("char.def: %s is already defined at line %d", cols[0], r.NumLine)
		}
		l, err := strconv.Atoi(cols[3])
		if err != nil {
			return fmt.Errorf("char.def: %s is invalid number at line %d", cols[3], r.NumLine)
		}
		catinfo := &categoryInfo{
			catType:  catType,
			isInvoke: cols[1] != "0",
			isGroup:  cols[2] != "0",
			length:   l,
		}
		p.categories[catType] = catinfo
	}
	return nil
}

func (p *MeCabOovProviderPlugin) readOov(unkDef string, grammar *dictionary.Grammar) error {
	var unkDefReader io.Reader
	if unkDef != "" {
		unkDefFd, err := os.OpenFile(unkDef, os.O_RDONLY, 0644)
		if err != nil {
			return err
		}
		defer unkDefFd.Close()
		unkDefReader = unkDefFd
	} else {
		unkDefF, err := data.Assets.Open("unk.def")
		if err != nil {
			return err
		}
		defer unkDefF.Close()
		unkDefReader = unkDefF
	}

	r := lnreader.NewLineNumberReader(unkDefReader)
	for {
		line, err := r.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		cols := strings.Split(string(line), ",")
		if len(cols) < 10 {
			return fmt.Errorf("unk.def: invalid format at line %d", r.NumLine)
		}
		catType, err := dictionary.GetCategoryType(cols[0])
		if err != nil {
			return fmt.Errorf("unk.def: %s is invalid type at line %d", cols[0], r.NumLine)
		}
		_, ok := p.categories[catType]
		if !ok {
			return fmt.Errorf("unk.def: %s is undefined at line %d", cols[0], r.NumLine)
		}

		leftId, err := strconv.ParseInt(cols[1], 10, 16)
		if err != nil {
			return fmt.Errorf("unk.def: %s is invalid number at line %d", cols[1], r.NumLine)
		}
		rightId, err := strconv.ParseInt(cols[2], 10, 16)
		if err != nil {
			return fmt.Errorf("unk.def: %s is invalid number at line %d", cols[2], r.NumLine)
		}
		cost, err := strconv.ParseInt(cols[3], 10, 16)
		if err != nil {
			return fmt.Errorf("unk.def: %s is invalid number at line %d", cols[3], r.NumLine)
		}
		pos := []string{cols[4], cols[5], cols[6], cols[7], cols[8], cols[9]}
		posId := grammar.GetPartOfSpeechId(pos)
		if posId == -1 {
			return fmt.Errorf("unk.def: unknown Part Of Speech at line %d", r.NumLine)
		}
		poov := &oov{
			leftId:  int16(leftId),
			rightId: int16(rightId),
			cost:    int16(cost),
			posId:   posId,
		}

		l, ok := p.oovList[catType]
		if !ok {
			ll := []*oov{}
			l = &ll
			p.oovList[catType] = l
		}
		*l = append(*l, poov)
	}
	return nil
}
