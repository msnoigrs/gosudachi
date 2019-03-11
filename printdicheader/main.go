package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/msnoigrs/gosudachi/dictionary"
	"github.com/msnoigrs/gosudachi/internal/mmap"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage of %s:
	%s file
`, os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	err := printHeader(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func printHeader(dictfile string) error {
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

	dh := dictionary.ParseDictionaryHeader(bytebuffer, 0)

	fmt.Println("filename:", dictfile)

	switch dh.Version {
	case dictionary.SystemDictVersion:
		fmt.Println("type: system dictionary")
	case dictionary.UserDictVersion:
		fmt.Println("type: user dictionary")
	default:
		fmt.Println("invalid file")
		os.Exit(1)
	}

	ctime := time.Unix(dh.CreateTime, 0)
	zone, _ := ctime.Zone()
	fmt.Printf("createTime: %s[%s]\n", ctime.Format(time.RFC3339), zone)
	fmt.Println("description:", dh.Description)

	return nil
}
