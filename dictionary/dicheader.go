package dictionary

import (
	"bytes"
	"encoding/binary"
	"errors"
)

const (
	DescriptionSize   = 256
	HeaderStorageSize = 8 + 8 + DescriptionSize
)

type DictionaryHeader struct {
	Version     uint64
	CreateTime  int64
	Description string
}

func NewDictionaryHeader(version uint64, createTime int64, description string) *DictionaryHeader {
	return &DictionaryHeader{
		Version:     version,
		CreateTime:  createTime,
		Description: description,
	}
}

func ParseDictionaryHeader(input []byte, offset int) *DictionaryHeader {
	offset, version := bufferToUint64(input, offset)
	offset, createTime := bufferToInt64(input, offset)

	i := offset
	for ; i < HeaderStorageSize; i++ {
		if input[i] == 0 {
			break
		}
	}
	// UTF-8
	description := string(input[offset:i])

	return &DictionaryHeader{
		Version:     version,
		CreateTime:  createTime,
		Description: description,
	}
}

func (dh *DictionaryHeader) ToBytes() ([]byte, error) {
	desc := []byte(dh.Description)
	if len(desc) > DescriptionSize {
		return nil, errors.New("description is too long")
	}

	buf := bytes.NewBuffer(make([]byte, 0, HeaderStorageSize))
	err := binary.Write(buf, binary.LittleEndian, uint64(dh.Version))
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.LittleEndian, uint64(dh.CreateTime))
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(desc)
	if err != nil {
		return nil, err
	}

	if len(desc) < DescriptionSize {
		padding := make([]byte, DescriptionSize-len(desc))
		_, err = buf.Write(padding)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}
