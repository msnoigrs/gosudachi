package gosudachi

import (
	"unicode/utf8"

	"github.com/msnoigrs/gosudachi/dictionary"
)

type InputText struct {
	OriginalText             string
	ModifiedText             string
	Bytea                    []byte
	offsets                  []int
	byteIndexes              []int
	charCategories           []uint32
	charCategoryContinuities []int
	canBowList               []bool
}

func NewInputText(originalText string, modifiedText string, bytea []byte, offsets []int, byteIndexes []int, charCategories []uint32, charCategoryContinuities []int, canBowList []bool) *InputText {
	return &InputText{
		OriginalText:             originalText,
		ModifiedText:             modifiedText,
		Bytea:                    bytea,
		offsets:                  offsets,
		byteIndexes:              byteIndexes,
		charCategories:           charCategories,
		charCategoryContinuities: charCategoryContinuities,
		canBowList:               canBowList,
	}
}

func (t *InputText) GetText() string {
	return t.ModifiedText
}

func (t *InputText) GetByteText() []byte {
	return t.Bytea
}

func (t *InputText) GetSubstring(begin int, end int) string {
	return string([]rune(t.ModifiedText)[t.byteIndexes[begin]:t.byteIndexes[end]])
}

func (t *InputText) GetOffsetTextLength(index int) int {
	return t.byteIndexes[index]
}

func (t *InputText) GetOriginalIndex(index int) int {
	return t.offsets[index]
}

func (t *InputText) GetCharCategoryTypes(index int) uint32 {
	return t.charCategories[t.byteIndexes[index]]
}

func (t *InputText) GetCharCategoryTypesRange(begin int, end int) uint32 {
	if begin+t.charCategoryContinuities[begin] < end {
		return uint32(0)
	}
	b := t.byteIndexes[begin]
	e := t.byteIndexes[end]
	continuousCategory := t.charCategories[b]
	for i := b + 1; i < e; i++ {
		continuousCategory &= t.charCategories[i]
	}
	return continuousCategory
}

func (t *InputText) GetCharCategoryContinuousLength(index int) int {
	return t.charCategoryContinuities[index]
}

func (t *InputText) GetCodePointsOffsetLength(index int, codePointOffset int) int {
	length := 0
	target := t.byteIndexes[index] + codePointOffset
	for i := index; i < len(t.Bytea); i++ {
		if t.byteIndexes[i] >= target {
			return length
		}
		length++
	}
	return length
}

func (t *InputText) CodePointCount(begin int, end int) int {
	return t.byteIndexes[end] - t.byteIndexes[begin]
}

func (t *InputText) CanBow(index int) bool {
	return t.IsCharAlignment(index) && t.canBowList[t.byteIndexes[index]]
}

func (t *InputText) IsCharAlignment(index int) bool {
	return (t.Bytea[index] & 0xC0) != 0x80
}

type InputTextBuilder struct {
	OriginalText  string
	modifiedRunes []rune
	textOffsets   []int
	grammar       *dictionary.Grammar
}

func NewInputTextBuilder(text string, grammar *dictionary.Grammar) *InputTextBuilder {
	modifiedRunes := []rune(text)
	offsetslen := len(modifiedRunes) + 1
	textOffsets := make([]int, offsetslen, offsetslen)
	for i := 0; i < len(modifiedRunes); i++ {
		textOffsets[i] = i
	}
	textOffsets[len(modifiedRunes)] = len(modifiedRunes)
	return &InputTextBuilder{
		OriginalText:  text,
		modifiedRunes: modifiedRunes,
		textOffsets:   textOffsets,
		grammar:       grammar,
	}
}

func (builder *InputTextBuilder) GetText() []rune {
	ret := make([]rune, len(builder.modifiedRunes))
	copy(ret, builder.modifiedRunes)
	return ret
}

func (builder *InputTextBuilder) Replace(begin int, end int, runes []rune) {
	rl := len(runes)
	tlen := end - begin

	offset := builder.textOffsets[begin]

	if rl < tlen {
		ol := len(builder.modifiedRunes)
		copy(builder.modifiedRunes[begin+rl:], builder.modifiedRunes[end:])
		copy(builder.modifiedRunes[begin:], runes)
		builder.modifiedRunes = builder.modifiedRunes[:ol-tlen+rl]

		tolen := len(builder.textOffsets)
		copy(builder.textOffsets[begin+rl:], builder.textOffsets[end:])
		builder.textOffsets = builder.textOffsets[:tolen-tlen+rl]
	} else if rl == tlen {
		copy(builder.modifiedRunes[begin:], runes)
	} else {
		builder.modifiedRunes = append(builder.modifiedRunes, make([]rune, rl-tlen)...)
		copy(builder.modifiedRunes[begin+rl:], builder.modifiedRunes[end:])
		copy(builder.modifiedRunes[begin:], runes)

		builder.textOffsets = append(builder.textOffsets, make([]int, rl-tlen)...)
		copy(builder.textOffsets[begin+rl:], builder.textOffsets[end:])
	}

	for i := 0; i < rl; i++ {
		builder.textOffsets[begin+i] = offset
	}
}

func (builder *InputTextBuilder) Build() *InputText {
	// getCharCategoryTypes
	runeCount := len(builder.modifiedRunes)
	charCategoryTypes := make([]uint32, runeCount, runeCount)
	for i := 0; i < runeCount; i++ {
		charCategoryTypes[i] = builder.grammar.CharCategory.GetCategoryTypes(builder.modifiedRunes[i])
	}

	modifiedText := string(builder.modifiedRunes)
	p := []byte(modifiedText)
	keepp := p
	bytelength := len(p)
	size := bytelength + 1
	indexes := make([]int, size, size)
	offsets := make([]int, size, size)

	sizes := make([]int, runeCount, runeCount)

	pi := 0
	for i := 0; len(p) > 0; i++ {
		_, size := utf8.DecodeRune(p)
		sizes[i] = size
		for j := 0; j < size; j++ {
			indexes[pi] = i
			offsets[pi] = builder.textOffsets[i]
			pi++
		}
		p = p[size:]
	}
	indexes[bytelength] = runeCount
	offsets[bytelength] = builder.textOffsets[len(builder.textOffsets)-1]

	// getCharCategoryContinuities
	charCategoryContinuities := make([]int, bytelength, bytelength)
	pi = 0
	for i := 0; i < runeCount; {
		next := i + getCharCategoryContinuousLength(charCategoryTypes, i)
		var length int
		for j := i; j < next; j++ {
			length += sizes[j]
		}
		for k := length; k > 0; k-- {
			charCategoryContinuities[pi] = k
			pi++
		}
		i = next
	}

	// buildCanBowList
	canBowList := make([]bool, runeCount, runeCount)
	if runeCount > 0 {
		canBowList[0] = true
		for i := 1; i < runeCount; i++ {
			types := charCategoryTypes[i]
			if (types&dictionary.ALPHA == dictionary.ALPHA) ||
				(types&dictionary.GREEK == dictionary.GREEK) ||
				(types&dictionary.CYRILLIC == dictionary.CYRILLIC) {
				cc := charCategoryTypes[i-1] & types
				canBowList[i] = cc == 0
				continue
			}
			canBowList[i] = true
		}
	}

	return &InputText{
		builder.OriginalText,
		modifiedText,
		keepp,
		offsets,
		indexes,
		charCategoryTypes,
		charCategoryContinuities,
		canBowList,
	}
}

func getCharCategoryContinuousLength(charCategories []uint32, offset int) int {
	continuousCategory := charCategories[offset]
	var length int
	for length = 1; length < len(charCategories)-offset; length++ {
		cc := continuousCategory & charCategories[offset+length]
		if cc == 0 {
			return length
		}
	}
	return length
}
