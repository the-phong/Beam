// phase05-corpus generates a public replay corpus or verifies one without keys.
package main

import (
	"flag"
	"fmt"
	"os"

	"example.com/pluto/internal/phase05/corpus"
)

func main() {
	mode := flag.String("mode", "verify", "generate or verify")
	path := flag.String("file", "research/corpus/phase_05/public_corpus.json", "public corpus path")
	flag.Parse()
	if err := run(*mode, *path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(mode, path string) error {
	switch mode {
	case "generate":
		c, err := corpus.Generate()
		if err != nil {
			return err
		}
		if err := corpus.CheckOracle(c); err != nil {
			return err
		}
		if err := corpus.WriteFresh(path, c); err != nil {
			return err
		}
	case "verify":
		c, hash, err := corpus.Load(path)
		if err != nil {
			return err
		}
		if err := corpus.CheckOracle(c); err != nil {
			return err
		}
		fmt.Printf("corpus_sha256=%s profiles=%d b0_cases=%d b1_cases=%d\n", hash, len(c.Profiles), len(c.Profiles[0].Cases), len(c.Profiles[1].Cases))
	default:
		return fmt.Errorf("unknown mode %q", mode)
	}
	return nil
}
