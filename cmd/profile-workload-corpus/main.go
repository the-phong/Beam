package main

import (
	"fmt"
	"os"

	"example.com/pluto/internal/profiles/load"
)

func main() {
	if len(os.Args) != 3 || (os.Args[1] != "generate" && os.Args[1] != "verify") {
		fmt.Fprintln(os.Stderr, "usage: profile-workload-corpus generate|verify FILE")
		os.Exit(2)
	}
	var err error
	if os.Args[1] == "generate" {
		var c load.ScenarioCorpus
		c, err = load.BuildScenarioCorpus()
		if err == nil {
			err = load.WriteScenarioCorpus(os.Args[2], c)
		}
	} else {
		_, err = load.ReadScenarioCorpus(os.Args[2])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("S06 workload corpus verified")
}
