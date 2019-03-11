package lnreader

import (
	"bufio"
	"io"
)

type LineNumberReader struct {
	r         *bufio.Reader
	rawBuffer []byte
	NumLine   int
}

func NewLineNumberReader(r io.Reader) *LineNumberReader {
	return &LineNumberReader{
		r: bufio.NewReader(r),
	}
}

func (r *LineNumberReader) ReadLine() ([]byte, error) {
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
		r.NumLine++
	}
	return line, err
}

func IsSkipLine(l []byte) bool {
	for i, c := range l {
		if i == 0 && c == '#' {
			return true
		} else {
			if c != ' ' && c != '\n' && c != '\t' {
				return false
			}
		}
	}
	return true
}

func IsEmptyLine(l []byte) bool {
	for _, c := range l {
		if c != ' ' && c != '\n' && c != '\t' {
			return false
		}
	}
	return true
}
