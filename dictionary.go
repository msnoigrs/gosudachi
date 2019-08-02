package gosudachi

import (
	"fmt"
	"io"
	"os"

	"github.com/msnoigrs/gosudachi/data"
	"github.com/msnoigrs/gosudachi/dictionary"
)

const (
	UserDictCostParMorph = -20
)

const maxcost = int(int16(^uint16(0) >> 1))
const mincost = int(-maxcost - 1)

type JapaneseDictionary struct {
	grammar            *dictionary.Grammar
	lexicon            *dictionary.LexiconSet
	inputTextPlugins   []InputTextPlugin
	oovProviderPlugins []OovProviderPlugin
	pathRewritePlugins []PathRewritePlugin
	dictionaries       []*dictionary.BinaryDictionary
}

func NewJapaneseDictionary(config *BaseConfig, inputTextPlugins []InputTextPlugin, oovProviderPlugins []OovProviderPlugin, pathRewritePlugins []PathRewritePlugin, editConnectionCostPlugins []EditConnectionCostPlugin) (*JapaneseDictionary, error) {
	if len(oovProviderPlugins) == 0 {
		return nil, fmt.Errorf("no OOV provider")
	}

	d := &JapaneseDictionary{
		inputTextPlugins:   inputTextPlugins,
		oovProviderPlugins: oovProviderPlugins,
		pathRewritePlugins: pathRewritePlugins,
	}

	err := d.ReadSystemDictionary(config.SystemDict, config.Utf16String)
	if err != nil {
		return nil, fmt.Errorf("fail to read a system dictionary: %s", err)
	}

	for _, plugin := range editConnectionCostPlugins {
		err := plugin.SetUp(d.grammar)
		if err != nil {
			return nil, err
		}
		err = plugin.Edit(d.grammar)
		if err != nil {
			return nil, err
		}
	}

	err = d.ReadCharacterDefinition(config.CharacterDefinitionFile)
	if err != nil {
		return nil, fmt.Errorf("fail to read a character defition file: %s", err)
	}

	for _, plugin := range inputTextPlugins {
		err := plugin.SetUp()
		if err != nil {
			return nil, err
		}
	}
	for _, plugin := range oovProviderPlugins {
		err := plugin.SetUp(d.grammar)
		if err != nil {
			return nil, err
		}
	}
	for _, plugin := range pathRewritePlugins {
		err := plugin.SetUp(d.grammar)
		if err != nil {
			return nil, err
		}
	}

	for _, ud := range config.UserDict {
		err := d.ReadUserDictionary(ud, config.Utf16String)
		if err != nil {
			return nil, fmt.Errorf("fail to read a user dictionary: %s", err)
		}
	}
	return d, nil
}

func (d *JapaneseDictionary) ReadSystemDictionary(filename string, utf16string bool) error {
	dict, err := dictionary.ReadSystemDictionary(filename, utf16string)
	if err != nil {
		return err
	}

	d.dictionaries = append(d.dictionaries, dict)
	d.grammar = dict.Grammar
	d.lexicon = dictionary.NewLexiconSet(dict.Lexicon)
	return nil
}

func (d *JapaneseDictionary) ReadUserDictionary(filename string, utf16string bool) error {
	if d.lexicon.IsFull() {
		return fmt.Errorf("too many dictionaries")
	}

	dict, err := dictionary.ReadUserDictionary(filename, utf16string)
	if err != nil {
		return err
	}

	d.dictionaries = append(d.dictionaries, dict)

	userLexicon := dict.Lexicon
	tokenizer := NewJapaneseTokenizer(
		d.grammar,
		d.lexicon,
		d.inputTextPlugins,
		d.oovProviderPlugins,
		[]PathRewritePlugin{},
	)
	userLexicon.CalculateCost(func(text string) (int16, error) {
		ms, err := tokenizer.Tokenize("C", text)
		if err != nil {
			return int16(mincost), err
		}
		cost := ms.GetInternalCost() + UserDictCostParMorph*ms.Length()
		if cost > maxcost {
			cost = maxcost
		} else if cost < mincost {
			cost = mincost
		}
		return int16(cost), nil
	})
	d.lexicon.Add(userLexicon, int32(d.grammar.GetPartOfSpeechSize()))
	d.grammar.AddPosList(dict.Grammar)
	return nil
}

func (d *JapaneseDictionary) ReadCharacterDefinition(charDef string) error {
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

	cat := dictionary.NewCharacterCategory()
	err := cat.ReadCharacterDefinition(charDefReader)
	if err != nil {
		return err
	}
	d.grammar.CharCategory = cat
	return nil
}

func (d *JapaneseDictionary) Close() {
	d.grammar = nil
	d.lexicon = nil
	for _, dict := range d.dictionaries {
		dict.Close()
	}
	d.dictionaries = d.dictionaries[:0]
}

func (d *JapaneseDictionary) Create() *JapaneseTokenizer {
	return NewJapaneseTokenizer(
		d.grammar,
		d.lexicon,
		d.inputTextPlugins,
		d.oovProviderPlugins,
		d.pathRewritePlugins,
	)
}

func (d *JapaneseDictionary) GetPartOfSpeechSize() int {
	return d.grammar.GetPartOfSpeechSize()
}

func (d *JapaneseDictionary) GetPartOfSpeechString(posId int16) []string {
	return d.grammar.GetPartOfSpeechString(posId)
}
