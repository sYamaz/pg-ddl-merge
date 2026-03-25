package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sYamaz/pg-ddl-merge/merger"
)

func main() {
	input := flag.String("input", ".", "input directory containing SQL files")
	flag.StringVar(input, "i", ".", "input directory containing SQL files (shorthand)")

	output := flag.String("output", "./merged.sql", "output file path")
	flag.StringVar(output, "o", "./merged.sql", "output file path (shorthand)")

	separator := flag.String("separator", "", "(deprecated) ignored in semantic merge mode")

	flag.Parse()

	count, err := merger.Run(merger.Config{
		InputDir:          *input,
		OutputPath:        *output,
		SeparatorTemplate: *separator,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Merged %d files into %s\n", count, *output)
}
