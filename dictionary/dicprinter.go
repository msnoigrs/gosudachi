package dictionary

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/msnoigrs/gosudachi/internal/mmap"
)

func PrintDictionary(filename string, utf16string bool, systemDict *BinaryDictionary, output io.Writer) error {
	var grammar *Grammar

	dic, err := NewBinaryDictionary(filename, utf16string)
	if err != nil {
		return err
	}
	defer dic.Close()
	if dic.Header.Version == SystemDictVersion {
		grammar = dic.Grammar
	} else if systemDict == nil {
		return errors.New("the system dictionary is not specified")
	} else {
		grammar = systemDict.Grammar
		if dic.Header.Version == UserDictVersion2 {
			grammar.AddPosList(dic.Grammar)
		}
	}

	possize := grammar.GetPartOfSpeechSize()
	posStrings := make([]string, possize, possize)
	for pid := 0; pid < possize; pid++ {
		posStrings = append(posStrings, strings.Join(grammar.GetPartOfSpeechString(int16(pid)), ","))
	}

	lexicon := dic.Lexicon
	for wordId := int32(0); wordId < lexicon.Size(); wordId++ {
		leftId := lexicon.GetLeftId(wordId)
		rightId := lexicon.GetRightId(wordId)
		cost := lexicon.GetCost(wordId)
		wi := lexicon.GetWordInfo(wordId)

		unitType := getUnitType(wi)

		fmt.Fprintf(output,
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
	return nil
}

func wordIdToString(wid int) string {
	if wid < 0 {
		return "*"
	}
	return strconv.Itoa(wid)
}

func getUnitType(wi *WordInfo) string {
	if len(wi.AUnitSplit) == 0 {
		return "A"
	} else if len(wi.BUnitSplit) == 0 {
		return "B"
	}
	return "C"
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

func PrintHeader(dictfile string, output io.Writer) error {
	dictfd, err := os.OpenFile(dictfile, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer dictfd.Close()

	finfo, err := dictfd.Stat()
	if err != nil {
		return err
	}

	bytebuffer, err := mmap.Mmap(dictfd, false, 0, finfo.Size())
	if err != nil {
		return err
	}
	defer mmap.Munmap(bytebuffer)

	dh := ParseDictionaryHeader(bytebuffer, 0)

	fmt.Fprintf(output, "filename: %s\n", dictfile)

	switch dh.Version {
	case SystemDictVersion:
		fmt.Fprintln(output, "type: system dictionary")
	case UserDictVersion, UserDictVersion2:
		fmt.Fprintln(output, "type: user dictionary")
	default:
		fmt.Fprintln(output, "invalid file")
		os.Exit(1)
	}

	ctime := time.Unix(dh.CreateTime, 0)
	zone, _ := ctime.Zone()
	fmt.Fprintf(output, "createTime: %s[%s]\n", ctime.Format(time.RFC3339), zone)
	fmt.Fprintf(output, "description: %s\n", dh.Description)

	return nil
}
