package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/msnoigrs/gosudachi/dictionary"
	"github.com/msnoigrs/gosudachi/internal/mmap"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage of %s:
	%s -o file -s file [-d description] [-j] file1 [file2 ...]

Options:
`, os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}

	var (
		outputpath  string
		systemdict  string
		description string
		utf16string bool
	)
	flag.StringVar(&outputpath, "o", "", "output to file")
	flag.StringVar(&systemdict, "s", "", "system dictionary")
	flag.StringVar(&description, "d", "", "comment")
	flag.BoolVar(&utf16string, "j", false, "use UTF-16 string")

	flag.Parse()

	if outputpath == "" || systemdict == "" || len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	dh := dictionary.NewDictionaryHeader(
		dictionary.UserDictVersion,
		time.Now().Unix(),
		description,
	)

	hb, err := dh.ToBytes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	sysdic, err := os.OpenFile(systemdict, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	defer sysdic.Close()

	finfo, err := sysdic.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	bytebuffer, err := mmap.Mmap(sysdic, false, 0, finfo.Size())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	defer mmap.Munmap(bytebuffer)

	sysdh := dictionary.ParseDictionaryHeader(bytebuffer, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	if sysdh.Version != dictionary.SystemDictVersion {
		fmt.Fprintf(os.Stderr, "Error: invalid system dictionary: %s", systemdict)
		os.Exit(1)
	}

	grammar := dictionary.NewGrammar(bytebuffer, dictionary.HeaderStorageSize, utf16string)

	outputWriter, err := os.OpenFile(outputpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	defer outputWriter.Close()

	bufout := bufio.NewWriter(outputWriter)
	n, err := bufout.Write(hb)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	err = bufout.Flush()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	dicbuilder := dictionary.NewDictionaryBuilder(int64(n), utf16string)

	fmt.Fprint(os.Stderr, "reading the source file...")
	for _, lexiconpath := range flag.Args() {
		err := build(dicbuilder, grammar, lexiconpath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s", err, lexiconpath)
			os.Exit(1)
		}
	}
	p := message.NewPrinter(language.English)
	p.Fprintf(os.Stderr, " %d words\n", dicbuilder.WordSize)

	err = dicbuilder.WriteLexicon(outputWriter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
}

func build(dicbuilder *dictionary.DictionaryBuilder, store dictionary.PosIdStore, lexiconpath string) error {
	lexiconReader, err := os.OpenFile(lexiconpath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer lexiconReader.Close()

	err = dicbuilder.BuildLexicon(store, lexiconReader)
	if err != nil {
		return err
	}
	return nil
}
