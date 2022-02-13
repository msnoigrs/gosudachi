package dictionary

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/msnoigrs/gosudachi/dartsclone"
	"github.com/msnoigrs/gosudachi/internal/lnreader"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	StringUtf8MaxLength  = 32767
	StringUtf16MaxLength = 32767
	ArrayMaxLength       = 127
	NumberOfColumns      = 19
	BufferSize           = 1024 * 1024
)

type wordEntry struct {
	Headword            string
	Parameters          [3]int16
	WordInfo            *WordInfo
	AUnitSplitString    string
	BUnitSplitString    string
	WordStructureString string
}

type PosIdStore interface {
	GetPosId(posstrings ...string) int16
	GetPartOfSpeechSize() int
}

type PosTable struct {
	table    []string
	contains map[string]int16
}

func (pt *PosTable) getId(s string) int16 {
	id, ok := pt.contains[s]
	if ok {
		return id
	}
	id = int16(len(pt.table))
	pt.contains[s] = id
	pt.table = append(pt.table, s)
	return id
}

func (pt *PosTable) GetPosId(posstrings ...string) int16 {
	return pt.getId(strings.Join(posstrings, ","))
}

func (pt *PosTable) GetPartOfSpeechSize() int {
	return len(pt.table)
}

func NewPosTable() *PosTable {
	return &PosTable{
		contains: map[string]int16{},
	}
}

type PosTableUser struct {
	PosTable
	baseStore PosIdStore
}

func (pt *PosTableUser) getId(s string) int16 {
	posId := pt.baseStore.GetPosId(s)
	if posId < 0 {
		posId = pt.PosTable.getId(s) + int16(pt.baseStore.GetPartOfSpeechSize())
	}
	return posId
}

func (pt *PosTableUser) GetPosId(posstrings ...string) int16 {
	return pt.getId(strings.Join(posstrings, ","))
}

func NewPosTableUser(base PosIdStore) *PosTableUser {
	return &PosTableUser{
		PosTable: PosTable{
			contains: map[string]int16{},
		},
		baseStore: base,
	}
}

type lexiconReader struct {
	r           *bufio.Reader
	rawBuffer   []byte
	numLine     int
	fieldBuffer []byte
}

func newLexiconReader(r io.Reader) *lexiconReader {
	return &lexiconReader{
		r: bufio.NewReader(r),
	}
}

func (r *lexiconReader) readLine() ([]byte, error) {
	line, err := r.r.ReadSlice('\n')
	if err == bufio.ErrBufferFull {
		r.rawBuffer = append(r.rawBuffer[:0], line...)
		for err == bufio.ErrBufferFull {
			line, err = r.r.ReadSlice('\n')
			r.rawBuffer = append(r.rawBuffer, line...)
		}
		line = r.rawBuffer
	}
	if len(line) > 0 && err == io.EOF {
		err = nil
	} else if err == nil {
		n := len(line)
		if n >= 2 && line[n-2] == '\r' && line[n-1] == '\n' {
			line = line[:n-2]
		} else {
			line = line[:n-1]
		}
	}
	if err == nil {
		r.numLine++
	}
	return line, err
}

func (r *lexiconReader) readRecord(dst []string) ([]string, error) {
	line, err := r.readLine()
	if err != nil {
		return nil, err
	}

	dst = dst[:0]

	var i int
	for {
		i = bytes.IndexByte(line, ',')
		field := line
		if i >= 0 {
			field = field[:i]
		}

		dst = append(dst, string(r.decode(field)))

		if i >= 0 {
			line = line[i+1:]
		} else {
			break
		}
	}

	return dst, nil
}

func (r *lexiconReader) decode(s []byte) []byte {
	r.fieldBuffer = r.fieldBuffer[:0]

	runecount := 0
	umark := -1
	nmark := -1
	brace := false
	ncount := 0

	numstart := -1
	last := 0
	for i := 0; i < len(s); {
		rc, width := utf8.DecodeRune(s[i:])
		runecount++
		i += width
		switch rc {
		case '\\':
			if nmark >= 0 {
				nmark = -1
			} else {
				r.fieldBuffer = append(r.fieldBuffer, s[last:i-width]...)
				last = i - width
				umark = runecount
			}
		case 'u':
			if nmark >= 0 {
				nmark = -1
			} else if umark >= 0 {
				if umark == runecount-1 {
					nmark = runecount
					brace = false
					ncount = 0
					numstart = i
				}
				umark = -1
			}
		case '{':
			if nmark >= 0 {
				if brace == false && ncount == 0 {
					brace = true
					numstart = i
				} else {
					nmark = -1
				}
			}
		case '}':
			if nmark >= 0 {
				if brace && ncount >= 0 {
					i32, err := strconv.ParseInt(string(s[numstart:i-width]), 16, 32)
					if i32 <= 0x10FFFF && err == nil {
						r.fieldBuffer = append(r.fieldBuffer, []byte(string(i32))...)
						last = i
					}
					brace = false
				}
				nmark = -1
			}
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f', 'A', 'B', 'C', 'D', 'E', 'F':
			if nmark >= 0 {
				ncount++
				if !brace {
					if ncount == 4 {
						i32, err := strconv.ParseInt(string(s[numstart:i]), 16, 32)
						if i32 <= 0x10FFFF && err == nil {
							r.fieldBuffer = append(r.fieldBuffer, []byte(string(i32))...)
							last = i
						}
						nmark = -1
					}
				}
			}
		default:
			if umark >= 0 {
				umark = -1
			}
			if nmark >= 0 {
				nmark = -1
			}
		}
	}
	if last > 0 {
		return append(r.fieldBuffer, s[last:]...)
	}
	return s
}

type writeStringFunc func(buffer *bytes.Buffer, s string) error
type stringLenFunc func(s string) bool

type DictionaryBuilder struct {
	trieKeys      *redblacktree.Tree
	wordEntries   []*wordEntry
	buffer        *bytes.Buffer
	position      int64
	systemLexicon *DoubleArrayLexicon
	writeStringF  writeStringFunc
	stringLen     stringLenFunc
}

func NewDictionaryBuilder(position int64, systemLexicon *DoubleArrayLexicon, utf16string bool) *DictionaryBuilder {
	ret := &DictionaryBuilder{
		trieKeys: redblacktree.NewWith(func(a, b interface{}) int {
			l, _ := a.([]byte)
			r, _ := b.([]byte)
			min := len(l)
			if min > len(r) {
				min = len(r)
			}
			for i := 0; i < min; i++ {
				if l[i] != r[i] {
					return (int(l[i]) & 0xff) - (int(r[i]) & 0xff)
				}
			}
			return len(l) - len(r)
		}),
		systemLexicon: systemLexicon,
		buffer:        bytes.NewBuffer([]byte{}),
		position:      position,
	}
	if utf16string {
		ret.writeStringF = writeStringUtf16
		ret.stringLen = utf16CountInString
	} else {
		ret.writeStringF = writeString
		ret.stringLen = utf8CountInString
	}
	return ret
}

func (dicbuilder *DictionaryBuilder) BuildLexicon(store PosIdStore, input io.Reader) error {
	var recordBuf []string
	r := newLexiconReader(input)
	for {
		cols, err := r.readRecord(recordBuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// parseLine
		if len(cols) != NumberOfColumns {
			return fmt.Errorf("invalid format at line: columns length must be %d: at line %d", NumberOfColumns, r.numLine)
		}

		if dicbuilder.stringLen(cols[0]) {
			return fmt.Errorf("string is too long: column 0 at line %d", r.numLine)
		}
		if dicbuilder.stringLen(cols[4]) {
			return fmt.Errorf("string is too long: column 4 at line %d", r.numLine)
		}
		if dicbuilder.stringLen(cols[11]) {
			return fmt.Errorf("string is too long: column 11 at line %d", r.numLine)
		}
		if dicbuilder.stringLen(cols[12]) {
			return fmt.Errorf("string is too long: column 12 at line %d", r.numLine)
		}

		entry := &wordEntry{}

		// headword for trie
		if cols[1] != "-1" {
			entry.Headword = cols[0]
		}

		// left-id, right-id, cost
		cols1, err := strconv.ParseInt(cols[1], 10, 16)
		if err != nil {
			return fmt.Errorf("%s: column 1 at line %d", err, r.numLine)
		}
		cols2, err := strconv.ParseInt(cols[2], 10, 16)
		if err != nil {
			return fmt.Errorf("%s: column 2 at line %d", err, r.numLine)
		}
		cols3, err := strconv.ParseInt(cols[3], 10, 16)
		if err != nil {
			return fmt.Errorf("%s: column 3 at line %d", err, r.numLine)
		}
		entry.Parameters[0] = int16(cols1)
		entry.Parameters[1] = int16(cols2)
		entry.Parameters[2] = int16(cols3)

		// part of speech
		posId := store.GetPosId(cols[5], cols[6], cols[7], cols[8], cols[9], cols[10])

		if strings.Count(cols[15], "/")+1 > ArrayMaxLength {
			return fmt.Errorf("too many units: columns 15 at line %d", r.numLine)
		}
		if strings.Count(cols[16], "/")+1 > ArrayMaxLength {
			return fmt.Errorf("too many units: columns 16 at line %d", r.numLine)
		}
		if strings.Count(cols[17], "/")+1 > ArrayMaxLength {
			return fmt.Errorf("too many units: columns 17 at line %d", r.numLine)
		}
		if cols[14] == "A" && (cols[15] != "*" || cols[16] != "*") {
			return fmt.Errorf("invalid splitting at line %d", r.numLine)
		}
		entry.AUnitSplitString = cols[15]
		entry.BUnitSplitString = cols[16]
		entry.WordStructureString = cols[17]

		var dicFormWordId int32
		if cols[13] == "*" {
			dicFormWordId = -1
		} else {
			cols13, err := strconv.ParseInt(cols[13], 10, 32)
			if err != nil {
				return fmt.Errorf("%s: column 13 at line %d", err, r.numLine)
			}
			dicFormWordId = int32(cols13)
		}

		entry.WordInfo = &WordInfo{
			Surface:              cols[4], // headword
			HeadwordLength:       int16(len(cols[0])),
			PosId:                posId,
			NormalizedForm:       cols[12],      // normalizedForm
			DictionaryFormWordId: dicFormWordId, // dictionaryFormWordId
			DictionaryForm:       "",            // dummy
			ReadingForm:          cols[11],      // readingForm
		}

		if entry.Headword != "" {
			// addToTrie
			wordId := int32(len(dicbuilder.wordEntries))
			v, ok := dicbuilder.trieKeys.Get([]byte(entry.Headword))
			if !ok {
				dicbuilder.trieKeys.Put([]byte(entry.Headword), []int32{wordId})
			} else {
				wordIds, _ := v.([]int32)
				dicbuilder.trieKeys.Put([]byte(entry.Headword), append(wordIds, wordId))
			}
		}
		dicbuilder.wordEntries = append(dicbuilder.wordEntries, entry)
	}

	return nil
}

func writeStringLength(buffer *bytes.Buffer, length int16) error {
	if length <= 127 {
		err := buffer.WriteByte(byte(length))
		if err != nil {
			return err
		}
	} else {
		err := buffer.WriteByte(byte((length >> 8) | 0x80))
		if err != nil {
			return err
		}
		err = buffer.WriteByte(byte(length & 0xFF))
		if err != nil {
			return err
		}
	}
	return nil
}

func writeString(buffer *bytes.Buffer, s string) error {
	err := writeStringLength(buffer, int16(len(s)))
	if err != nil {
		return err
	}
	_, err = buffer.WriteString(s)
	if err != nil {
		return err
	}
	return err
}

func writeStringUtf16(buffer *bytes.Buffer, s string) error {
	// java compatible
	javainternal := utf16.Encode([]rune(s))

	err := writeStringLength(buffer, int16(len(javainternal)))
	if err != nil {
		return err
	}
	for _, c := range javainternal {
		err = binary.Write(buffer, binary.LittleEndian, c)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeIntArray(buffer *bytes.Buffer, a []int32) error {
	err := buffer.WriteByte(byte(len(a)))
	if err != nil {
		return err
	}
	for _, i := range a {
		err := binary.Write(buffer, binary.LittleEndian, uint32(i))
		if err != nil {
			return err
		}
	}
	return nil
}

func (dicbuilder *DictionaryBuilder) writeString(s string) error {
	return dicbuilder.writeStringF(dicbuilder.buffer, s)
}

func (dicbuilder *DictionaryBuilder) writeIntArray(a []int32) error {
	return writeIntArray(dicbuilder.buffer, a)
}

func (dicbuilder *DictionaryBuilder) WriteGrammar(postable *PosTable, input io.Reader, writer io.Writer) error {
	bwriter := bufio.NewWriter(writer)

	fmt.Fprint(os.Stderr, "writing the POS table...")

	err := dicbuilder.convertPOSTable(postable)
	if err != nil {
		return err
	}
	n, err := dicbuilder.buffer.WriteTo(bwriter)
	if err != nil {
		return err
	}
	dicbuilder.position += n
	p := message.NewPrinter(language.English)
	p.Fprintf(os.Stderr, " %d bytes\n", n)
	dicbuilder.buffer.Reset()

	// convertMatrix
	r := lnreader.NewLineNumberReader(input)
	header, err := r.ReadLine()
	if err == io.EOF {
		return fmt.Errorf("invalid format at line %d", r.NumLine)
	}

	fmt.Fprint(os.Stderr, "writing the connection matrix...")

	lr := strings.Fields(string(header))
	if len(lr) < 2 {
		return fmt.Errorf("invalid format at line %d", r.NumLine)
	}
	leftSize, err := strconv.ParseInt(lr[0], 10, 16)
	if err != nil {
		return fmt.Errorf("%s: invalid format at line %d", err, r.NumLine)
	}
	rightSize, err := strconv.ParseInt(lr[1], 10, 16)
	if err != nil {
		return fmt.Errorf("%s: invalid format at line %d", err, r.NumLine)
	}
	err = binary.Write(dicbuilder.buffer, binary.LittleEndian, uint16(leftSize))
	if err != nil {
		return fmt.Errorf("%s: invalid format at line %d", err, r.NumLine)
	}
	err = binary.Write(dicbuilder.buffer, binary.LittleEndian, uint16(rightSize))
	if err != nil {
		return fmt.Errorf("%s: invalid format at line %d", err, r.NumLine)
	}

	n, err = dicbuilder.buffer.WriteTo(bwriter)
	if err != nil {
		return err
	}
	dicbuilder.position += n
	dicbuilder.buffer.Reset()

	buflen := 2 * leftSize * rightSize
	matrix := make([]byte, buflen, buflen)

	for {
		line, err := r.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if lnreader.IsEmptyLine(line) {
			continue
		}
		cols := strings.Fields(string(line))
		if len(cols) < 3 {
			fmt.Fprintf(os.Stderr, "invalid format at line %d\n", r.NumLine)
			continue
		}
		left, err := strconv.ParseInt(cols[0], 10, 16)
		if err != nil {
			return fmt.Errorf("%s: invalid format at line %d", err, r.NumLine)
		}
		right, err := strconv.ParseInt(cols[1], 10, 16)
		if err != nil {
			return fmt.Errorf("%s: invalid format at line %d", err, r.NumLine)
		}
		cost, err := strconv.ParseInt(cols[2], 10, 16)
		if err != nil {
			return fmt.Errorf("%s: invalid format at line %d", err, r.NumLine)
		}
		binary.LittleEndian.PutUint16(matrix[2*(left+leftSize*right):], uint16(cost))
	}

	nm, err := bwriter.Write(matrix)
	if err != nil {
		return err
	}
	dicbuilder.position += int64(nm)
	p.Fprintf(os.Stderr, " %d bytes\n", nm+4)

	err = bwriter.Flush()
	if err != nil {
		return err
	}

	return nil
}

func (dicbuilder *DictionaryBuilder) WriteGrammarUser(postable *PosTable, writer io.Writer) error {
	bwriter := bufio.NewWriter(writer)

	fmt.Fprint(os.Stderr, "writing the POS table...")

	err := dicbuilder.convertPOSTable(postable)
	if err != nil {
		return err
	}
	n, err := dicbuilder.buffer.WriteTo(bwriter)
	if err != nil {
		return err
	}
	dicbuilder.position += n
	p := message.NewPrinter(language.English)
	p.Fprintf(os.Stderr, " %d bytes\n", n)
	dicbuilder.buffer.Reset()

	fmt.Fprint(os.Stderr, "writing the connection matrix...")

	err = binary.Write(dicbuilder.buffer, binary.LittleEndian, uint16(0))
	if err != nil {
		return err
	}
	err = binary.Write(dicbuilder.buffer, binary.LittleEndian, uint16(0))
	if err != nil {
		return err
	}

	n, err = dicbuilder.buffer.WriteTo(bwriter)
	if err != nil {
		return err
	}
	dicbuilder.position += 4
	fmt.Fprint(os.Stderr, " 4 bytes\n")
	dicbuilder.buffer.Reset()

	err = bwriter.Flush()
	if err != nil {
		return err
	}

	return nil
}

func (dicbuilder *DictionaryBuilder) convertPOSTable(postable *PosTable) error {
	err := binary.Write(dicbuilder.buffer, binary.LittleEndian, uint16(len(postable.table)))
	if err != nil {
		return err
	}

	for _, pos := range postable.table {
		ts := strings.Split(pos, ",")
		for _, t := range ts {
			err := dicbuilder.writeString(t)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (dicbuilder *DictionaryBuilder) WriteLexicon(writer io.WriteSeeker, store PosIdStore) error {
	bwriter := bufio.NewWriter(writer)

	trie := dartsclone.NewDoubleArray()

	size := dicbuilder.trieKeys.Size()

	keys := make([][]byte, size, size)
	values := make([]int, size, size)
	wordIdTable := bytes.NewBuffer(make([]byte, 0, len(dicbuilder.wordEntries)*(4+2)))

	var position int
	it := dicbuilder.trieKeys.Iterator()
	for i := 0; it.Next(); i++ {
		k := it.Key()
		v := it.Value()
		bkey, _ := k.([]byte)
		wordIds, _ := v.([]int32)
		keys[i] = bkey
		values[i] = position
		err := wordIdTable.WriteByte(byte(len(wordIds)))
		if err != nil {
			return err
		}
		position++
		for _, wid := range wordIds {
			err := binary.Write(wordIdTable, binary.LittleEndian, uint32(wid))
			if err != nil {
				return err
			}
			position += 4
		}
	}

	fmt.Fprint(os.Stderr, "building the trie")

	err := trie.Build(keys, values, func(state int, max int) {
		if state%((max/10)+1) == 0 {
			fmt.Fprint(os.Stderr, ".")
		}
	})
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "done")

	fmt.Fprint(os.Stderr, "writing the trie...")
	dicbuilder.buffer.Reset()

	err = binary.Write(dicbuilder.buffer, binary.LittleEndian, uint32(trie.Length()))
	if err != nil {
		return err
	}
	n, err := dicbuilder.buffer.WriteTo(bwriter)
	if err != nil {
		return err
	}
	dicbuilder.position += n
	dicbuilder.buffer.Reset()

	nn, err := bwriter.Write(trie.ByteArray())
	if err != nil {
		return err
	}
	dicbuilder.position += int64(nn)
	p := message.NewPrinter(language.English)
	p.Fprintf(os.Stderr, " %d bytes\n", nn+4)

	fmt.Fprint(os.Stderr, "writing the word-ID table...")
	err = binary.Write(dicbuilder.buffer, binary.LittleEndian, uint32(position))
	if err != nil {
		return err
	}
	n, err = dicbuilder.buffer.WriteTo(bwriter)
	if err != nil {
		return err
	}
	dicbuilder.position += n
	dicbuilder.buffer.Reset()

	n, err = wordIdTable.WriteTo(bwriter)
	if err != nil {
		return err
	}
	dicbuilder.position += n
	p.Fprintf(os.Stderr, " %d bytes\n", n+4)

	fmt.Fprint(os.Stderr, "writing the word parameters...")
	err = binary.Write(dicbuilder.buffer, binary.LittleEndian, uint32(len(dicbuilder.wordEntries)))
	if err != nil {
		return err
	}
	for _, entry := range dicbuilder.wordEntries {
		err = binary.Write(dicbuilder.buffer, binary.LittleEndian, uint16(entry.Parameters[0]))
		if err != nil {
			return err
		}
		err = binary.Write(dicbuilder.buffer, binary.LittleEndian, uint16(entry.Parameters[1]))
		if err != nil {
			return err
		}
		err = binary.Write(dicbuilder.buffer, binary.LittleEndian, uint16(entry.Parameters[2]))
		if err != nil {
			return err
		}
		n, err = dicbuilder.buffer.WriteTo(bwriter)
		if err != nil {
			return err
		}
		dicbuilder.position += n
		dicbuilder.buffer.Reset()
	}
	p.Fprintf(os.Stderr, " %d bytes\n", len(dicbuilder.wordEntries)*6+4)

	err = bwriter.Flush()
	if err != nil {
		return err
	}

	err = dicbuilder.writeWordInfo(writer, store)
	if err != nil {
		return err
	}

	return nil
}

func (dicbuilder *DictionaryBuilder) writeWordInfo(writer io.WriteSeeker, store PosIdStore) error {
	offsetslen := int64(4 * len(dicbuilder.wordEntries))
	_, err := writer.Seek(offsetslen, io.SeekCurrent)
	if err != nil {
		return err
	}
	bwriter := bufio.NewWriter(writer)

	offsets := bytes.NewBuffer(make([]byte, 0, offsetslen))

	fmt.Fprint(os.Stderr, "writing the wordInfos...")
	base := dicbuilder.position + offsetslen
	position := base
	for _, we := range dicbuilder.wordEntries {
		wi := we.WordInfo
		err = binary.Write(offsets, binary.LittleEndian, uint32(position))
		if err != nil {
			return err
		}
		err = dicbuilder.writeString(wi.Surface)
		if err != nil {
			return err
		}
		err = writeStringLength(dicbuilder.buffer, wi.HeadwordLength)
		if err != nil {
			return err
		}
		err := binary.Write(dicbuilder.buffer, binary.LittleEndian, uint16(wi.PosId))
		if err != nil {
			return err
		}
		var normalizedForm string
		if wi.NormalizedForm != wi.Surface {
			normalizedForm = wi.NormalizedForm
		}
		err = dicbuilder.writeString(normalizedForm)
		if err != nil {
			return err
		}
		err = binary.Write(dicbuilder.buffer, binary.LittleEndian, uint32(wi.DictionaryFormWordId))
		if err != nil {
			return err
		}
		var readingForm string
		if wi.ReadingForm != wi.Surface {
			readingForm = wi.ReadingForm
		}
		err = dicbuilder.writeString(readingForm)
		if err != nil {
			return err
		}
		aUnitSplit, err := dicbuilder.parseSplitInfo(we.AUnitSplitString, store)
		if err != nil {
			return err
		}
		err = dicbuilder.writeIntArray(aUnitSplit)
		if err != nil {
			return err
		}
		bUnitSplit, err := dicbuilder.parseSplitInfo(we.BUnitSplitString, store)
		if err != nil {
			return err
		}
		err = dicbuilder.writeIntArray(bUnitSplit)
		if err != nil {
			return err
		}
		wordStructure, err := dicbuilder.parseSplitInfo(we.WordStructureString, store)
		if err != nil {
			return err
		}
		err = dicbuilder.writeIntArray(wordStructure)
		if err != nil {
			return err
		}
		n, err := dicbuilder.buffer.WriteTo(bwriter)
		if err != nil {
			return err
		}
		dicbuilder.buffer.Reset()
		position += n
	}
	p := message.NewPrinter(language.English)
	p.Fprintf(os.Stderr, " %d bytes\n", position-base)
	err = bwriter.Flush()
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stderr, "writing wordInfo offsets...")
	_, err = writer.Seek(dicbuilder.position, io.SeekStart)
	if err != nil {
		return err
	}
	bwriter = bufio.NewWriter(writer)
	n, err := offsets.WriteTo(bwriter)
	if err != nil {
		return err
	}
	p.Fprintf(os.Stderr, " %d bytes\n", n)
	err = bwriter.Flush()
	if err != nil {
		return err
	}

	return nil
}

func (dicbuilder *DictionaryBuilder) parseSplitInfo(info string, store PosIdStore) ([]int32, error) {
	if info == "*" {
		return []int32{}, nil
	}
	words := strings.Split(info, "/")
	if len(words) > ArrayMaxLength {
		return []int32{}, errors.New("too many units")
	}
	ret := make([]int32, 0, len(words))
	for _, word := range words {
		if strings.HasPrefix(word, "U") {
			parsed, err := strconv.ParseInt(word[1:], 10, 32)
			if err == nil {
				pint := int32(parsed)
				if pint < 0 || pint >= int32(len(dicbuilder.wordEntries)) {
					return []int32{}, fmt.Errorf("invalid word ID: %s", word)
				}
				if dicbuilder.systemLexicon != nil {
					pint |= (int32(1) << 28)
				}
				ret = append(ret, pint)
				continue
			}
		}

		parsed, err := strconv.ParseInt(word, 10, 32)
		if err == nil {
			pint := int32(parsed)
			if dicbuilder.systemLexicon != nil {
				if pint < 0 || pint >= dicbuilder.systemLexicon.Size() {
					return []int32{}, fmt.Errorf("invalid word ID: %s", word)
				}
			} else {
				if pint < 0 || pint >= int32(len(dicbuilder.wordEntries)) {
					return []int32{}, fmt.Errorf("invalid word ID: %s", word)
				}
			}
			ret = append(ret, pint)
			continue
		}

		wid, err := dicbuilder.wordToId(word, store)
		if err != nil {
			return []int32{}, err
		}
		if wid < 0 {
			return []int32{}, fmt.Errorf("not found such a word: %s", word)
		}
		ret = append(ret, int32(wid))
	}
	return ret, nil
}

func (dicbuilder *DictionaryBuilder) wordToId(text string, store PosIdStore) (int32, error) {
	var recordBuf []string
	r := newLexiconReader(strings.NewReader(text))
	for {
		cols, err := r.readRecord(recordBuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return int32(-1), err
		}
		if len(cols) < 8 {
			return int32(-1), fmt.Errorf("too few columns: %s", text)
		}
		posId := store.GetPosId(cols[1], cols[2], cols[3], cols[4], cols[5], cols[6])
		if posId < 0 {
			return int32(-1), fmt.Errorf("invalid part of speech: %s", text)
		}
		return dicbuilder.getWordId(cols[0], posId, cols[7]), nil
	}
	return int32(-1), nil
}

func (dicbuilder *DictionaryBuilder) getWordId(headword string, posId int16, readingForm string) int32 {
	for wid, entry := range dicbuilder.wordEntries {
		wi := entry.WordInfo
		if wi.Surface == headword && wi.PosId == posId && wi.ReadingForm == readingForm {
			if dicbuilder.systemLexicon != nil {
				return int32(wid) | (int32(1) << 28)
			}
			return int32(wid)
		}
	}
	if dicbuilder.systemLexicon != nil {
		return dicbuilder.systemLexicon.GetWordId(headword, posId, readingForm)
	}
	return int32(-1)
}

func (dicbuilder *DictionaryBuilder) EntrySize() int {
	return len(dicbuilder.wordEntries)
}

func utf16CountInString(s string) bool {
	return len(utf16.Encode([]rune(s))) > StringUtf16MaxLength
}

func utf8CountInString(s string) bool {
	return len(s) > StringUtf8MaxLength
}
