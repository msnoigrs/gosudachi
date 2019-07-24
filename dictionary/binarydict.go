package dictionary

import (
	"fmt"
	"github.com/msnoigrs/gosudachi/internal/mmap"
	"os"
)

type BinaryDictionary struct {
	fd      *os.File
	fmap    []byte
	Header  *DictionaryHeader
	Grammar *Grammar
	Lexicon *DoubleArrayLexicon
}

func NewBinaryDictionary(filename string, utf16string bool) (*BinaryDictionary, error) {
	fd, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}

	finfo, err := fd.Stat()
	if err != nil {
		_ = fd.Close()
		return nil, err
	}
	fmap, err := mmap.Mmap(fd, false, 0, finfo.Size())
	if err != nil {
		_ = fd.Close()
		return nil, err
	}

	offset := 0
	header := ParseDictionaryHeader(fmap, offset)
	if header == nil {
		return nil, fmt.Errorf("invalid header: %s", filename)
	}

	offset += HeaderStorageSize
	var grammar *Grammar
	if header.Version == SystemDictVersion {
		grammar = NewGrammar(fmap, offset, utf16string)
		offset += grammar.StorageSize
	} else if header.Version != UserDictVersion {
		_ = mmap.Munmap(fmap)
		_ = fd.Close()
		return nil, fmt.Errorf("invalid dictionary: %s", filename)
	}

	lexicon := NewDoubleArrayLexicon(fmap, offset, utf16string)

	return &BinaryDictionary{
		fd,
		fmap,
		header,
		grammar,
		lexicon,
	}, nil
}

func ReadSystemDictionary(filename string, utf16string bool) (*BinaryDictionary, error) {
	dict, err := NewBinaryDictionary(filename, utf16string)
	if err != nil {
		return nil, err
	}
	if dict.Header.Version != SystemDictVersion {
		_ = dict.Close()
		return nil, fmt.Errorf("invalid systemd dictionary: %s", filename)
	}
	return dict, nil
}

func ReadUserDictionary(filename string, utf16string bool) (*BinaryDictionary, error) {
	dict, err := NewBinaryDictionary(filename, utf16string)
	if err != nil {
		return nil, err
	}
	if dict.Header.Version != UserDictVersion {
		_ = dict.Close()
		return nil, fmt.Errorf("invalid user dictionary: %s", filename)
	}
	return dict, nil
}

func (bd *BinaryDictionary) Close() error {
	err := mmap.Munmap(bd.fmap)
	if err != nil {
		return err
	}
	return bd.fd.Close()
}
