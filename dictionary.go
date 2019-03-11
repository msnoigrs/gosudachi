package gosudachi

import (
	"fmt"
	"io"
	"os"

	"github.com/msnoigrs/gosudachi/data"
	"github.com/msnoigrs/gosudachi/dictionary"
	"github.com/msnoigrs/gosudachi/internal/mmap"
)

const (
	UserDictCostParMorph = -20
)

const maxcost = int(int16(^uint16(0) >> 1))
const mincost = int(-maxcost - 1)

type dicFile struct {
	fd   *os.File
	fmap []byte
}

func newDicFile(filename string) (*dicFile, error) {
	fd, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}

	finfo, err := fd.Stat()
	if err != nil {
		fd.Close()
		return nil, err
	}
	fmap, err := mmap.Mmap(fd, false, 0, finfo.Size())
	if err != nil {
		fd.Close()
		return nil, err
	}
	return &dicFile{
		fd:   fd,
		fmap: fmap,
	}, nil
}

func (d *dicFile) munmap() error {
	err := mmap.Munmap(d.fmap)
	err = d.fd.Close()
	return err
}

type JapaneseDictionary struct {
	grammar            *dictionary.Grammar
	lexicon            *dictionary.LexiconSet
	inputTextPlugins   []InputTextPlugin
	oovProviderPlugins []OovProviderPlugin
	pathRewritePlugins []PathRewritePlugin
	buffers            []*dicFile
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
	df, err := newDicFile(filename)
	if err != nil {
		return err
	}

	offset := 0
	header := dictionary.ParseDictionaryHeader(df.fmap, offset)
	if header == nil {
		df.munmap()
		return fmt.Errorf("invalid header: %s", filename)
	}
	if header.Version != dictionary.SystemDictVersion {
		df.munmap()
		return fmt.Errorf("invalid system dictionary: %s", filename)
	}
	offset += dictionary.HeaderStorageSize

	d.grammar = dictionary.NewGrammar(df.fmap, offset, utf16string)
	offset += d.grammar.StorageSize

	d.lexicon = dictionary.NewLexiconSet(dictionary.NewDoubleArrayLexicon(df.fmap, offset, utf16string))

	d.buffers = append(d.buffers, df)
	return nil
}

func (d *JapaneseDictionary) ReadUserDictionary(filename string, utf16string bool) error {
	if d.lexicon.IsFull() {
		return fmt.Errorf("too many dictionaries")
	}

	df, err := newDicFile(filename)
	if err != nil {
		return err
	}

	offset := 0
	header := dictionary.ParseDictionaryHeader(df.fmap, offset)
	if header == nil {
		df.munmap()
		return fmt.Errorf("invalid header: %s", filename)
	}
	if header.Version != dictionary.UserDictVersion {
		df.munmap()
		return fmt.Errorf("invalid user dictionary: %s", filename)
	}
	offset += dictionary.HeaderStorageSize

	userLexicon := dictionary.NewDoubleArrayLexicon(df.fmap, offset, utf16string)
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
	d.lexicon.Add(userLexicon)
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
	for _, df := range d.buffers {
		df.munmap()
	}
	d.buffers = d.buffers[:0]
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
