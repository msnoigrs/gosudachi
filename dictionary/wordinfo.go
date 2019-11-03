package dictionary

type WordInfo struct {
	Surface              string
	HeadwordLength       int16
	PosId                int16
	NormalizedForm       string
	DictionaryFormWordId int32
	DictionaryForm       string
	ReadingForm          string
	AUnitSplit           []int32
	BUnitSplit           []int32
	WordStructure        []int32
}
