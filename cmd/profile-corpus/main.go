package main

import (
	"fmt"
	"os"

	"example.com/pluto/internal/profiles/corpus"
)

func main() {
	if len(os.Args) != 3 || (os.Args[1] != "generate" && os.Args[1] != "verify") {
		fmt.Fprintln(os.Stderr, "usage: profile-corpus generate|verify FILE")
		os.Exit(2)
	}
	var err error
	if os.Args[1] == "generate" {
		var c researchv2corpus.Corpus
		c, err = researchv2corpus.Build()
		if err == nil {
			err = researchv2corpus.Write(os.Args[2], c)
		}
	} else {
		_, err = researchv2corpus.Read(os.Args[2])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
