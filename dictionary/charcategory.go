package dictionary

import (
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/msnoigrs/gosudachi/internal/lnreader"
)

// Categories of characters
const (
	DEFAULT      uint32 = 1       // The fall back category
	SPACE        uint32 = 1 << 1  // WhiteSpaces
	KANJI        uint32 = 1 << 2  // CJKV ideographic characters
	SYMBOL       uint32 = 1 << 3  // Symbols
	NUMERIC      uint32 = 1 << 4  // Numerical characters
	ALPHA        uint32 = 1 << 5  // Latin alphabets
	HIRAGANA     uint32 = 1 << 6  // Hiragana characters
	KATAKANA     uint32 = 1 << 7  // Katakana characters
	KANJINUMERIC uint32 = 1 << 8  // Knaji numeric characters
	GREEK        uint32 = 1 << 9  // Greek alphabets
	CYRILLIC     uint32 = 1 << 10 // Cyrillic alphabets
	USER1        uint32 = 1 << 11 // User defined category
	USER2        uint32 = 1 << 12 // User defined category
	USER3        uint32 = 1 << 13 // User defined category
	USER4        uint32 = 1 << 14 // User defined category
	NOOOVBOW     uint32 = 1 << 15 // Characters that cannot be the beginning of word
)

func GetCategoryType(s string) (uint32, error) {
	switch s {
	case "DEFAULT":
		return DEFAULT, nil
	case "SPACE":
		return SPACE, nil
	case "KANJI":
		return KANJI, nil
	case "SYMBOL":
		return SYMBOL, nil
	case "NUMERIC":
		return NUMERIC, nil
	case "ALPHA":
		return ALPHA, nil
	case "HIRAGANA":
		return HIRAGANA, nil
	case "KATAKANA":
		return KATAKANA, nil
	case "KANJINUMERIC":
		return KANJINUMERIC, nil
	case "GREEK":
		return GREEK, nil
	case "CYRILLIC":
		return CYRILLIC, nil
	case "USER1":
		return USER1, nil
	case "USER2":
		return USER2, nil
	case "USER3":
		return USER3, nil
	case "USER4":
		return USER4, nil
	case "NOOOVBOW":
		return NOOOVBOW, nil
	}
	return 0, fmt.Errorf("%s is invalid type", s)
}

type categoryRange struct {
	low        int32
	high       int32
	categories uint32
}

func (r *categoryRange) contains(cp rune) bool {
	if int32(cp) >= r.low && int32(cp) <= r.high {
		return true
	}
	return false
}

func (r *categoryRange) containingLength(text string) int {
	for i, c := range text {
		if int32(c) < r.low || int32(c) > r.high {
			return i
		}
	}
	return utf8.RuneCountInString(text)
}

type CharacterCategory struct {
	rangeList []*categoryRange
}

func NewCharacterCategory() *CharacterCategory {
	return &CharacterCategory{}
}

func (cc CharacterCategory) GetCategoryTypes(codePoint rune) uint32 {
	var categories uint32
	for _, cr := range cc.rangeList {
		if cr.contains(codePoint) {
			categories |= cr.categories
		}
	}

	if categories == 0 {
		categories = DEFAULT
	}
	return categories
}

func (cc CharacterCategory) ReadCharacterDefinition(charDefReader io.Reader) error {
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
			return fmt.Errorf("invalid format at line %d: too short fields", r.NumLine)
		}
		if !strings.HasPrefix(cols[0], "0x") {
			continue
		}

		catrange := new(categoryRange)
		rs := strings.Split(cols[0], "..")
		low, err := decodeHexStrToInt32(rs[0])
		if err != nil {
			return fmt.Errorf("invalid format at line %d: %s", r.NumLine, err)
		}
		catrange.low = low
		if len(rs) > 1 {
			high, err := decodeHexStrToInt32(rs[1])
			if err != nil {
				return fmt.Errorf("invalid format at line %d: %s", r.NumLine, err)
			}
			catrange.high = high
		} else {
			catrange.high = catrange.low
		}
		if catrange.low > catrange.high {
			return fmt.Errorf("invalid format at line %d: low > high", r.NumLine)
		}
		for i := 1; i < len(cols); i++ {
			if strings.HasPrefix(cols[i], "#") {
				break
			}
			t, err := GetCategoryType(cols[i])
			if err != nil {
				return fmt.Errorf("%s at line %d: %s", err, r.NumLine, err)
			}
			catrange.categories |= t
		}
		cc.rangeList = append(cc.rangeList, catrange)
	}

	return nil
}

func decodeHexStrToInt32(s string) (int32, error) {
	if len(s) < 3 {
		return 0, fmt.Errorf("invalid hex string: too short")
	}
	src := []byte(s[2:])
	dst := make([]byte, hex.DecodedLen(len(src)))
	n, err := hex.Decode(dst, src)
	if err != nil {
		return 0, err
	}
	if n > 4 {
		return 0, fmt.Errorf("invalid hex string: too long")
	}
	var ret int32
	switch n {
	case 4:
		ret = int32(dst[0])*33554432 + int32(dst[1])*131072 + int32(dst[2])*512 + int32(dst[3])
	case 3:
		ret = int32(dst[0])*131072 + int32(dst[1])*512 + int32(dst[2])
	case 2:
		ret = int32(dst[0])*512 + int32(dst[1])
	case 1:
		ret = int32(dst[0])
	}
	return ret, nil
}
