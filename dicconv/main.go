package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/msnoigrs/gosudachi/dictionary"
	"github.com/msnoigrs/gosudachi/internal/mmap"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type mmapFrom struct {
	header  *dictionary.DictionaryHeader
	grammar *dictionary.Grammar
	lexicon *dictionary.DoubleArrayLexicon
	fd      *os.File
	m       []byte
}

func (mg *mmapFrom) mclose() {
	if mg.header == nil {
		return
	}
	mmap.Munmap(mg.m)
	mg.fd.Close()
	mg.header = nil
}

func readDic(dictfile string, utf16string bool) (*mmapFrom, error) {
	dictfd, err := os.OpenFile(dictfile, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}

	finfo, err := dictfd.Stat()
	if err != nil {
		dictfd.Close()
		return nil, err
	}

	bytebuffer, err := mmap.Mmap(dictfd, false, 0, finfo.Size())
	if err != nil {
		dictfd.Close()
		return nil, err
	}

	offset := 0
	header := dictionary.ParseDictionaryHeader(bytebuffer, 0)
	if err != nil {
		mmap.Munmap(bytebuffer)
		dictfd.Close()
		return nil, err
	}
	offset += dictionary.HeaderStorageSize

	var grammar *dictionary.Grammar
	if header.Version == dictionary.SystemDictVersion {
		grammar = dictionary.NewGrammar(bytebuffer, offset, utf16string)
		offset += grammar.StorageSize
	} else if header.Version != dictionary.UserDictVersion {
		mmap.Munmap(bytebuffer)
		dictfd.Close()
		return nil, fmt.Errorf("file is invalid: %s", dictfile)
	}

	lexicon := dictionary.NewDoubleArrayLexicon(bytebuffer, offset, utf16string)

	return &mmapFrom{
		header:  header,
		grammar: grammar,
		lexicon: lexicon,
		fd:      dictfd,
		m:       bytebuffer,
	}, nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage of %s:
	%s [-o file] [-j] file

Options:
`, os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}

	var (
		outputfile  string
		utf16string bool
	)
	flag.StringVar(&outputfile, "o", "", "output to file")
	flag.BoolVar(&utf16string, "j", false, "from UTF-8 to UTF-16")

	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if outputfile == "" {
		if utf16string {
			outputfile = "out_utf16.dic"
		} else {
			outputfile = "out_utf8.dic"
		}
	}
	if !filepath.IsAbs(outputfile) {
		var err error
		outputfile, err = filepath.Abs(outputfile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	outputfd, err := os.OpenFile(outputfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", outputfile, err)
		os.Exit(1)
	}
	defer outputfd.Close()
	bufiooutput := bufio.NewWriter(outputfd)

	args := flag.Args()
	fromdic, err := readDic(args[0], !utf16string)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	hb, err := fromdic.header.ToBytes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	var offset int64
	n, err := bufiooutput.Write(hb)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to write header: %s", err)
		os.Exit(1)
	}
	offset = int64(n)

	var n64 int64
	p := message.NewPrinter(language.English)
	if fromdic.grammar != nil {
		fmt.Fprint(os.Stderr, "writting the POS table...")
		buffer := bytes.NewBuffer([]byte{})
		err = fromdic.grammar.WritePOSTableTo(buffer, utf16string)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		n64, err = buffer.WriteTo(bufiooutput)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		p.Fprintf(os.Stderr, " %d bytes\n", n64)
		buffer.Reset()
		offset += n64

		fmt.Fprint(os.Stderr, "writting the connection matrix...")
		n, err = fromdic.grammar.WriteConnMatrixTo(bufiooutput)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		p.Fprintf(os.Stderr, " %d bytes\n", n)
		offset += int64(n)
	}

	fmt.Fprint(os.Stderr, "writting the trie...")
	n, err = fromdic.lexicon.WriteTrieTo(bufiooutput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	p.Fprintf(os.Stderr, " %d bytes\n", n)
	offset += int64(n)

	fmt.Fprint(os.Stderr, "writting the word-ID table...")
	n, err = fromdic.lexicon.WriteWordIdTableTo(bufiooutput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	p.Fprintf(os.Stderr, " %d bytes\n", n)
	offset += int64(n)

	fmt.Fprint(os.Stderr, "writting the word parameters...")
	n, err = fromdic.lexicon.WriteWordParamsTo(bufiooutput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	p.Fprintf(os.Stderr, " %d bytes\n", n)
	offset += int64(n)

	err = bufiooutput.Flush()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	fmt.Fprint(os.Stderr, "writting the wordInfos...")
	offsetlen := int64(4 * fromdic.lexicon.Size())
	_, err = outputfd.Seek(offsetlen, io.SeekCurrent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	bufiooutput = bufio.NewWriter(outputfd)

	n, offsets, err := fromdic.lexicon.WriteWordInfos(bufiooutput, offset, offsetlen, utf16string)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	p.Fprintf(os.Stderr, " %d bytes\n", n)

	err = bufiooutput.Flush()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	fmt.Fprint(os.Stderr, "writting wordInfo offsets...")
	_, err = outputfd.Seek(offset, io.SeekStart)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	bufiooutput = bufio.NewWriter(outputfd)

	n64, err = offsets.WriteTo(bufiooutput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	p.Fprintf(os.Stderr, " %d bytes\n", n64)

	err = bufiooutput.Flush()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
}
