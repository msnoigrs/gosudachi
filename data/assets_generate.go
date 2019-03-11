// +build ignore

package main

import (
	"log"

	"github.com/msnoigrs/gosudachi/data"
	"github.com/shurcooL/vfsgen"
)

func main() {
	err := vfsgen.Generate(data.Assets, vfsgen.Options{
		BuildTags:    "!dev",
		PackageName:  "data",
		VariableName: "Assets",
	})

	if err != nil {
		log.Fatalln(err)
	}
}
