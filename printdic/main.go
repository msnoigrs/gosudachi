package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/msnoigrs/gosudachi/dictionary"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage of %s:
	%s [-s file] [-j] file

Options:
`, os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}

	var (
		systemdict  string
		utf16string bool
	)
	flag.StringVar(&systemdict, "s", "", "system dictionary")
	flag.BoolVar(&utf16string, "j", false, "use UTF-16 string")

	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	var (
		sdic *dictionary.BinaryDictionary
		err  error
	)
	if systemdict != "" {
		sdic, err = dictionary.ReadSystemDictionary(systemdict, utf16string)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer sdic.Close()
	}

	err = dictionary.PrintDictionary(flag.Args()[0], utf16string, sdic, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
