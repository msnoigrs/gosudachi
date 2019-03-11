package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/msnoigrs/gosudachi/dictionary"
	"github.com/msnoigrs/gosudachi/internal/mmap"
)

type mmapGrammar struct {
	grammar *dictionary.Grammar
	fd      *os.File
	m       []byte
}

func (mg *mmapGrammar) mclose() {
	if mg.grammar == nil {
		return
	}
	mmap.Munmap(mg.m)
	mg.fd.Close()
	mg.grammar = nil
}

func readGrammar(systemdict string, utf16string bool) (*mmapGrammar, error) {
	sysdic, err := os.OpenFile(systemdict, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}

	finfo, err := sysdic.Stat()
	if err != nil {
		sysdic.Close()
		return nil, err
	}

	bytebuffer, err := mmap.Mmap(sysdic, false, 0, finfo.Size())
	if err != nil {
		sysdic.Close()
		return nil, err
	}

	sysdh := dictionary.ParseDictionaryHeader(bytebuffer, 0)
	if err != nil {
		mmap.Munmap(bytebuffer)
		sysdic.Close()
		return nil, err
	}
	if sysdh.Version != dictionary.SystemDictVersion {
		mmap.Munmap(bytebuffer)
		sysdic.Close()
		return nil, fmt.Errorf("%s is not a system dictionary", systemdict)
	}

	grammar := dictionary.NewGrammar(bytebuffer, dictionary.HeaderStorageSize, utf16string)

	return &mmapGrammar{
		grammar: grammar,
		fd:      sysdic,
		m:       bytebuffer,
	}, nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage of %s:
	%s [-s file] [-j] file

Options:
`, os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}

	var (
		systemdict  string
		utf16string bool
	)
	flag.StringVar(&systemdict, "s", "", "system dictionary")
	flag.BoolVar(&utf16string, "j", false, "use UTF-16 string")

	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	var (
		mg  *mmapGrammar
		err error
	)
	if systemdict != "" {
		mg, err = readGrammar(systemdict, utf16string)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		defer mg.mclose()
	}

	dic, err := os.OpenFile(flag.Args()[0], os.O_RDONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	defer dic.Close()

	finfo, err := dic.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	bytebuffer, err := mmap.Mmap(dic, false, 0, finfo.Size())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	defer mmap.Munmap(bytebuffer)

	offset := 0
	dh := dictionary.ParseDictionaryHeader(bytebuffer, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	offset += dictionary.HeaderStorageSize

	var grammar *dictionary.Grammar
	if dh.Version == dictionary.SystemDictVersion {
		if mg != nil {
			mg.mclose()
		}
		grammar = dictionary.NewGrammar(bytebuffer, offset, utf16string)
		offset += grammar.StorageSize
	} else {
		if mg == nil {
			fmt.Fprintf(os.Stderr, "the system dictionary is not specified")
			os.Exit(1)
		}
		grammar = mg.grammar
	}
	possize := grammar.GetPartOfSpeechSize()
	posStrings := make([]string, possize, possize)
	for pid := 0; pid < possize; pid++ {
		posStrings = append(posStrings, strings.Join(grammar.GetPartOfSpeechString(int16(pid)), ","))
	}

	lexicon := dictionary.NewDoubleArrayLexicon(bytebuffer, offset, utf16string)
	for wordId := int32(0); wordId < lexicon.Size(); wordId++ {
		leftId := lexicon.GetLeftId(wordId)
		rightId := lexicon.GetRightId(wordId)
		cost := lexicon.GetCost(wordId)
		wi := lexicon.GetWordInfo(wordId)

		unitType := getUnitType(wi)

		fmt.Printf(
			"%s,%d,%d,%d,%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
			wi.Surface,
			leftId,
			rightId,
			cost,
			wi.Surface,
			posStrings[int(wi.PosId)],
			wi.ReadingForm,
			wi.NormalizedForm,
			wordIdToString(int(wi.DictionaryFormWordId)),
			unitType,
			splitToString(wi.AUnitSplit),
			splitToString(wi.BUnitSplit),
			splitToString(wi.WordStructure),
		)
	}
}

func getUnitType(wi *dictionary.WordInfo) string {
	if len(wi.AUnitSplit) == 0 {
		return "A"
	} else if len(wi.BUnitSplit) == 0 {
		return "B"
	}
	return "C"
}

func wordIdToString(wid int) string {
	if wid < 0 {
		return "*"
	}
	return strconv.Itoa(wid)
}

func splitToString(split []int32) string {
	if len(split) == 0 {
		return "*"
	}
	splitstrs := make([]string, len(split), len(split))
	for _, i := range split {
		splitstrs = append(splitstrs, strconv.Itoa(int(i)))
	}
	return strings.Join(splitstrs, "/")
}
